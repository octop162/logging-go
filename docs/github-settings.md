# GitHub リポジトリ設定

リポジトリ: [octop162/logging-go](https://github.com/octop162/logging-go)

以下のコマンドで設定を再現できる。`gh auth login` 済みであること。

## デフォルトブランチを `main` に変更

```bash
# ローカルブランチのリネーム
git branch -m master main
git push -u origin main

# GitHub上のデフォルトブランチ変更（Settings > Default branch で手動変更）
# 旧masterブランチの削除
git push origin --delete master
```

## マージ後のブランチ自動削除

```bash
gh api repos/octop162/logging-go -X PATCH --input - <<'EOF'
{
  "delete_branch_on_merge": true
}
EOF
```

## ブランチ保護ルール（`main`）

```bash
gh api repos/octop162/logging-go/branches/main/protection -X PUT --input - <<'EOF'
{
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": false,
    "require_code_owner_reviews": false,
    "required_approving_review_count": 0
  },
  "enforce_admins": true,
  "required_status_checks": null,
  "restrictions": null
}
EOF
```

設定内容:
- PR必須（承認0人でもOK）
- 管理者にも適用
- Force push 禁止
- ブランチ削除禁止

## 設定の確認

```bash
# ブランチ保護ルールの確認
gh api repos/octop162/logging-go/branches/main/protection

# リポジトリ設定の確認（delete_branch_on_merge等）
gh api repos/octop162/logging-go --jq '{default_branch, delete_branch_on_merge}'
```
