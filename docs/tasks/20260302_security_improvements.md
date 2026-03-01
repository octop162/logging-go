# セキュリティ改善タスク

2026-03-02 セキュリティ監査で確認された未解消事項の対応。

## チェックリスト

- [ ] govulncheck を CI（build.yml）に組み込む
- [ ] gopsutil v3 → v4 移行（internal/monitor/process.go のインポートパス変更）
- [ ] COM vtable オフセット（internal/monitor/chrome.go）の妥当性を Windows メジャーアップデート時に確認する仕組みの検討

## 備考

- 間接依存の古いパッケージ（lufia/plan9stats, power-devops/perfstat 等）は gopsutil v4 移行で自動解消
- 2026-03-02 時点で govulncheck 既知CVE: 0件
