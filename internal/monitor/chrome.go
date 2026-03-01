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
	vtblFindFirst         uintptr = 5  // IUnknown(3) + SetFocus,GetRuntimeId
	vtblGetCurrentPattern uintptr = 16 // IUnknown(3) + 13
)

// IUIAutomationValuePattern vtable index
const (
	vtblGetCurrentValue uintptr = 4 // IUnknown(3) + SetValue
)

// UI Automation 定数
const (
	uiaValuePatternID        = 10002
	uiaControlTypePropertyID = 30003
	uiaEditControlTypeID     = 50004
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

// GetChromeURL は Chrome ウィンドウのアドレスバーから URL を取得する。
// アドレスバーが見つからない場合は空文字を返す。
func GetChromeURL(hwnd windows.HWND) (string, error) {
	// COM 初期化
	r1, _, _ := procCoInit.Call(0, coinitApartmentThreaded)
	switch uint32(r1) {
	case 0x00000000, 0x00000001: // S_OK, S_FALSE（同モードで既初期化）
		defer func() { _, _, _ = procCoUninit.Call() }()
	case 0x80010106: // RPC_E_CHANGED_MODE: 別モードで初期化済み、そのまま継続
	default:
		return "", fmt.Errorf("CoInitializeEx: HRESULT=0x%08X", uint32(r1))
	}

	// IUIAutomation インスタンス生成
	var uia uintptr
	r1, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidCUIAutomation)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIUIAutomation)),
		uintptr(unsafe.Pointer(&uia)),
	)
	if r1 != 0 {
		return "", fmt.Errorf("CoCreateInstance: HRESULT=0x%08X", uint32(r1))
	}
	defer comRelease(uia)

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
