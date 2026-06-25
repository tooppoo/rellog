
COVERAGE_THRESHOLD=90

set -eu -o pipefail

# ------------------------------------------------------------
# 1. 前回の coverage 実行結果を削除する
# ------------------------------------------------------------
rm -f coverage.out.tmp coverage.out coverage.func.tmp

# ------------------------------------------------------------
# 2. go test を実行する
#
# go test が失敗しても、ここでは処理を止めない。
# 終了コードだけ test_status に保存し、coverage の生成・表示・
# threshold 判定を後続で実行する。
# ------------------------------------------------------------
test_status=0
go test -covermode=atomic -coverpkg=./... -coverprofile=coverage.out.tmp ./... || test_status=$?

# ------------------------------------------------------------
# 3. coverage profile を整形する
#
# coverage.out.tmp が生成されていれば、main.go を除外して
# coverage.out を作成する。
#
# coverage profile の先頭行 `mode: atomic` は必須なので、
# NR == 1 で必ず残す。
#
# go test がコンパイルエラーなどで失敗した場合、
# coverage.out.tmp が存在しないことがある。
# その場合でも後続処理を続けるため、最低限の coverage.out を作る。
# ------------------------------------------------------------
if [ -f coverage.out.tmp ]; then
  awk 'NR == 1 || $0 !~ /(^|\/)main\.go:/' coverage.out.tmp > coverage.out
else
  printf 'mode: atomic\n' > coverage.out
fi

# ------------------------------------------------------------
# 4. coverage の詳細を出力する
#
# go tool cover の結果は、画面表示にも threshold 判定にも使うため、
# 一度 coverage.func.tmp に保存する。
# ------------------------------------------------------------
coverage_status=0
go tool cover -func="$(pwd)/coverage.out" > coverage.func.tmp || coverage_status=$?
cat coverage.func.tmp

# ------------------------------------------------------------
# 5. total coverage が threshold を満たすか判定する
#
# total 行から coverage percentage を取り出し、
# COVERAGE_THRESHOLD を下回っていれば coverage_status を失敗扱いにする。
# ------------------------------------------------------------
awk -v threshold="$COVERAGE_THRESHOLD" '
  /^total:/ {
    found = 1
    coverage = $3
    sub(/%/, "", coverage)

    if (coverage + 0 < threshold + 0) {
      printf("coverage %.1f%% is below %.1f%%\n", coverage, threshold)
      exit 1
    }

    printf("coverage %.1f%% meets %.1f%% threshold\n", coverage, threshold)
  }

  END {
    if (!found) {
      print "coverage total line was not found"
      exit 1
    }
  }
' coverage.func.tmp || coverage_status=$?

# ------------------------------------------------------------
# 6. 一時ファイルを削除する
# ------------------------------------------------------------
rm -f coverage.func.tmp

# ------------------------------------------------------------
# 7. 最終的な終了コードを決める
#
# 以下のどちらかに該当すれば coverage target は失敗にする。
#
# - go test が失敗した
# - coverage が threshold を下回った
# ------------------------------------------------------------
if [ "$test_status" -ne 0 ]; then
  printf 'go test failed with exit code %d\n' "$test_status"
fi

if [ "$test_status" -ne 0 ] || [ "$coverage_status" -ne 0 ]; then
  exit 1
fi
