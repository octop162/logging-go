package monitor

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

var (
	modOle32             = windows.NewLazySystemDLL("ole32.dll")
	modOleAut32          = windows.NewLazySystemDLL("oleaut32.dll")
	procCoCreateInstance = modOle32.NewProc("CoCreateInstance")
	procCoInit           = modOle32.NewProc("CoInitializeEx")
	procCoUninit         = modOle32.NewProc("CoUninitialize")
	procSysFreeString    = modOleAut32.NewProc("SysFreeString")
	procVariantClear     = modOleAut32.NewProc("VariantClear")

	// CUIAutomation: {FF48DBA4-60EF-4201-AA87-54103EEF594E}
	clsidCUIAutomation = ole.GUID{
		Data1: 0xFF48DBA4,
		Data2: 0x60EF,
		Data3: 0x4201,
		Data4: [8]byte{0xAA, 0x87, 0x54, 0x10, 0x3E, 0xEF, 0x59, 0x4E},
	}
	// IUIAutomation: {30CBE57D-D9D0-452A-AB13-7AC5AC4825EE}
	iidIUIAutomation = ole.GUID{
		Data1: 0x30CBE57D,
		Data2: 0xD9D0,
		Data3: 0x452A,
		Data4: [8]byte{0xAB, 0x13, 0x7A, 0xC5, 0xAC, 0x48, 0x25, 0xEE},
	}
)

// IUIAutomation vtable index
const (
	vtblElementFromHandle       uintptr = 6  // IUnknown(3) + CompareElements,CompareRuntimeIds,GetRootElement
	vtblCreatePropertyCondition uintptr = 23 // IUnknown(3) + 20
)

// IUIAutomationElement vtable index
const (
	vtblFindFirst               uintptr = 5  // IUnknown(3) + SetFocus,GetRuntimeId
	vtblFindAll                 uintptr = 6  // IUnknown(3) + SetFocus,GetRuntimeId,FindFirst
	vtblGetCurrentPropertyValue uintptr = 10 // IUnknown(3) + 7
	vtblGetCurrentPattern       uintptr = 16 // IUnknown(3) + 13
)

// IUIAutomationElementArray vtable index
const (
	vtblArrayLength     uintptr = 3 // IUnknown(3)
	vtblArrayGetElement uintptr = 4 // IUnknown(3) + get_Length
)

// IUIAutomationValuePattern vtable index
const (
	vtblGetCurrentValue uintptr = 4 // IUnknown(3) + SetValue
)

// UI Automation 定数
const (
	uiaValuePatternID        = 10002
	uiaNamePropertyID        = 30005
	uiaControlTypePropertyID = 30003
	uiaEditControlTypeID     = 50004
	uiaTabItemControlTypeID  = 50019
	uiaDocumentControlTypeID = 50030
	treeScopeSubtree         = 7
	clsctxInprocServer       = 1
	coinitApartmentThreaded  = 0x2
)

// vtblCall は COM オブジェクトの vtable[idx] を呼び出す。
// this ポインタは自動的に先頭に追加される。hr != 0 の場合はエラーを返す。
func vtblCall(obj uintptr, idx uintptr, args ...uintptr) error {
	vtable := *(*uintptr)(unsafe.Pointer(obj))                                //nolint:govet // COM vtable access requires unsafe pointer arithmetic
	fn := *(*uintptr)(unsafe.Pointer(vtable + idx*unsafe.Sizeof(uintptr(0)))) //nolint:govet // COM vtable access requires unsafe pointer arithmetic
	all := make([]uintptr, 0, 1+len(args))
	all = append(all, obj)
	all = append(all, args...)
	r1, _, _ := syscall.SyscallN(fn, all...)
	if r1 != 0 {
		return fmt.Errorf("HRESULT=0x%08X", uint32(r1))
	}
	return nil
}

// comRelease は COM オブジェクトの IUnknown::Release() を呼ぶ。
func comRelease(obj uintptr) {
	if obj == 0 {
		return
	}
	vtable := *(*uintptr)(unsafe.Pointer(obj))                              //nolint:govet // COM vtable access requires unsafe pointer arithmetic
	fn := *(*uintptr)(unsafe.Pointer(vtable + 2*unsafe.Sizeof(uintptr(0)))) //nolint:govet // COM vtable access requires unsafe pointer arithmetic
	_, _, _ = syscall.SyscallN(fn, obj)
}

// initCOM は COM を初期化し、IUIAutomation インスタンスを返す。
// 戻り値の cleanup を defer で呼ぶこと。
func initCOM() (uia uintptr, cleanup func(), err error) {
	r1, _, _ := procCoInit.Call(0, coinitApartmentThreaded)
	needUninit := false
	switch uint32(r1) {
	case 0x00000000, 0x00000001:
		needUninit = true
	case 0x80010106:
	default:
		return 0, nil, fmt.Errorf("CoInitializeEx: HRESULT=0x%08X", uint32(r1))
	}

	r1, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidCUIAutomation)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIUIAutomation)),
		uintptr(unsafe.Pointer(&uia)),
	)
	if r1 != 0 {
		if needUninit {
			_, _, _ = procCoUninit.Call()
		}
		return 0, nil, fmt.Errorf("CoCreateInstance: HRESULT=0x%08X", uint32(r1))
	}

	cleanup = func() {
		comRelease(uia)
		if needUninit {
			_, _, _ = procCoUninit.Call()
		}
	}
	return uia, cleanup, nil
}

// getElementName は UIA 要素の Name プロパティを返す。
func getElementName(elem uintptr) string {
	var v ole.VARIANT
	if err := vtblCall(elem, vtblGetCurrentPropertyValue, uintptr(uiaNamePropertyID), uintptr(unsafe.Pointer(&v))); err != nil {
		return ""
	}
	defer func() { _, _, _ = procVariantClear.Call(uintptr(unsafe.Pointer(&v))) }()
	if v.VT != ole.VT_BSTR {
		return ""
	}
	bp := *(*uintptr)(unsafe.Pointer(&v.Val)) //nolint:govet // VARIANT union access
	if bp == 0 {
		return ""
	}
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(bp))) //nolint:govet // BSTR to UTF16 conversion
}

// collectTabItemNames は指定要素のサブツリーから TabItem(50019) の Name を収集する。
func collectTabItemNames(uia, elem uintptr) []string {
	var v ole.VARIANT
	v.VT = ole.VT_I4
	v.Val = int64(uiaTabItemControlTypeID)
	var cond uintptr
	if err := vtblCall(uia, vtblCreatePropertyCondition,
		uintptr(uiaControlTypePropertyID),
		uintptr(unsafe.Pointer(&v)),
		uintptr(unsafe.Pointer(&cond)),
	); err != nil {
		return nil
	}
	defer comRelease(cond)

	var arr uintptr
	if err := vtblCall(elem, vtblFindAll, uintptr(treeScopeSubtree), cond, uintptr(unsafe.Pointer(&arr))); err != nil || arr == 0 {
		return nil
	}
	defer comRelease(arr)

	var count int32
	if err := vtblCall(arr, vtblArrayLength, uintptr(unsafe.Pointer(&count))); err != nil {
		return nil
	}

	names := make([]string, 0, count)
	for i := int32(0); i < count; i++ {
		var item uintptr
		if err := vtblCall(arr, vtblArrayGetElement, uintptr(i), uintptr(unsafe.Pointer(&item))); err != nil || item == 0 {
			continue
		}
		name := getElementName(item)
		comRelease(item)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// GetChromeTabTitles は Chrome ウィンドウの全ブラウザタブのタイトルを返す。
// ウェブページ内の Tab UI（YouTube のフィルタ等）は除外する。
//
// アルゴリズム:
//  1. root のサブツリーから全 TabItem(50019) の名前を取得
//  2. root のサブツリーから全 Document(50030) 要素を検索
//  3. 各 Document の子孫 TabItem の名前を「除外セット」に収集
//  4. 除外セットに含まれない TabItem のみ返す（= Chrome ブラウザタブ）
func GetChromeTabTitles(hwnd windows.HWND) ([]string, error) {
	uia, cleanup, err := initCOM()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var root uintptr
	if err := vtblCall(uia, vtblElementFromHandle, uintptr(hwnd), uintptr(unsafe.Pointer(&root))); err != nil {
		return nil, fmt.Errorf("ElementFromHandle: %w", err)
	}
	defer comRelease(root)

	// 1. root のサブツリーから全 TabItem(50019) を取得
	allNames := collectTabItemNames(uia, root)
	if len(allNames) == 0 {
		return nil, nil
	}

	// 2. root のサブツリーから全 Document(50030) 要素を取得
	var docVar ole.VARIANT
	docVar.VT = ole.VT_I4
	docVar.Val = int64(uiaDocumentControlTypeID)
	var docCond uintptr
	if err := vtblCall(uia, vtblCreatePropertyCondition,
		uintptr(uiaControlTypePropertyID),
		uintptr(unsafe.Pointer(&docVar)),
		uintptr(unsafe.Pointer(&docCond)),
	); err != nil {
		return allNames, nil
	}
	defer comRelease(docCond)

	var docArr uintptr
	if err := vtblCall(root, vtblFindAll, uintptr(treeScopeSubtree), docCond, uintptr(unsafe.Pointer(&docArr))); err != nil || docArr == 0 {
		// Document が無い → 全 TabItem が Chrome ブラウザタブ
		return allNames, nil
	}
	defer comRelease(docArr)

	// 3. 各 Document の子孫 TabItem 名を除外セットに収集
	exclude := make(map[string]struct{})
	var docCount int32
	if err := vtblCall(docArr, vtblArrayLength, uintptr(unsafe.Pointer(&docCount))); err != nil {
		return allNames, nil
	}

	for i := int32(0); i < docCount; i++ {
		var doc uintptr
		if err := vtblCall(docArr, vtblArrayGetElement, uintptr(i), uintptr(unsafe.Pointer(&doc))); err != nil || doc == 0 {
			continue
		}
		webNames := collectTabItemNames(uia, doc)
		comRelease(doc)
		for _, n := range webNames {
			exclude[n] = struct{}{}
		}
	}

	if len(exclude) == 0 {
		return allNames, nil
	}

	// 4. 除外セットに含まれないものだけ返す
	result := make([]string, 0, len(allNames))
	for _, n := range allNames {
		if _, excluded := exclude[n]; !excluded {
			result = append(result, n)
		}
	}
	return result, nil
}

// GetChromeURL は Chrome ウィンドウのアドレスバーから URL を取得する。
// アドレスバーが見つからない場合は空文字を返す。
func GetChromeURL(hwnd windows.HWND) (string, error) {
	uia, cleanup, err := initCOM()
	if err != nil {
		return "", err
	}
	defer cleanup()

	// ElementFromHandle: Chrome ウィンドウのルート要素を取得
	var elem uintptr
	if err := vtblCall(uia, vtblElementFromHandle, uintptr(hwnd), uintptr(unsafe.Pointer(&elem))); err != nil {
		return "", fmt.Errorf("ElementFromHandle: %w", err)
	}
	defer comRelease(elem)

	// CreatePropertyCondition: ControlType = Edit (50004) の条件を作成
	var variant ole.VARIANT
	variant.VT = ole.VT_I4
	variant.Val = int64(uiaEditControlTypeID)
	var cond uintptr
	if err := vtblCall(uia, vtblCreatePropertyCondition,
		uintptr(uiaControlTypePropertyID),
		uintptr(unsafe.Pointer(&variant)),
		uintptr(unsafe.Pointer(&cond)),
	); err != nil {
		return "", fmt.Errorf("CreatePropertyCondition: %w", err)
	}
	defer comRelease(cond)

	// FindFirst: サブツリーから最初の Edit 要素（アドレスバー）を検索
	var addrBar uintptr
	if err := vtblCall(elem, vtblFindFirst,
		uintptr(treeScopeSubtree),
		cond,
		uintptr(unsafe.Pointer(&addrBar)),
	); err != nil {
		return "", fmt.Errorf("FindFirst: %w", err)
	}
	if addrBar == 0 {
		return "", nil
	}
	defer comRelease(addrBar)

	// GetCurrentPattern: IUIAutomationValuePattern を取得
	var pattern uintptr
	if err := vtblCall(addrBar, vtblGetCurrentPattern,
		uintptr(uiaValuePatternID),
		uintptr(unsafe.Pointer(&pattern)),
	); err != nil {
		return "", fmt.Errorf("GetCurrentPattern: %w", err)
	}
	if pattern == 0 {
		return "", nil
	}
	defer comRelease(pattern)

	// get_CurrentValue: アドレスバーの値（URL）を BSTR として取得
	var bstr uintptr
	if err := vtblCall(pattern, vtblGetCurrentValue, uintptr(unsafe.Pointer(&bstr))); err != nil {
		return "", fmt.Errorf("get_CurrentValue: %w", err)
	}
	if bstr == 0 {
		return "", nil
	}
	url := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(bstr))) //nolint:govet // BSTR to UTF16 conversion requires unsafe pointer cast
	_, _, _ = procSysFreeString.Call(bstr)
	return url, nil
}
