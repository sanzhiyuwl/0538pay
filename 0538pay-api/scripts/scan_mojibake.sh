#!/usr/bin/env bash
# 全库乱码/坏字节自检脚本
# 用途：扫描 pay0538 库所有文本列里含 UTF-8 替换字符(U+FFFD, 字节 EF BF BD)的坏数据，
#       并核查库/表/列字符集是否统一 utf8mb4。坏数据成因见 CLAUDE.md / 记忆 db-mojibake-*。
# 用法：bash 0538pay-api/scripts/scan_mojibake.sh
# 前提：Git Bash。若 mysql 路径不同，改下面 MYSQL_BIN。

set -euo pipefail

MYSQL_BIN="/f/phpstudy_pro/Extensions/MySQL5.7.26/bin/mysql.exe"
DB="pay0538"
USER="pay0538"
PASS="pay0538"

MYSQL="$MYSQL_BIN -u$USER -p$PASS $DB --default-character-set=utf8mb4"

echo "=== [1/2] 扫描含替换字符(损坏)的 表.列.行数 ==="
# 对每个文本列生成一条检测语句，逐条执行，只输出命中项
gen=$($MYSQL -N -e "
SELECT CONCAT(
  'SELECT ''', table_name, ''' AS tbl, ''', column_name, ''' AS col, COUNT(*) AS bad FROM \`', table_name, '\` WHERE \`', column_name, '\` LIKE ''%', CHAR(0xEF,0xBF,0xBD), '%'' HAVING bad>0'
)
FROM information_schema.columns
WHERE table_schema='$DB'
  AND data_type IN ('char','varchar','text','tinytext','mediumtext','longtext');
" 2>/dev/null)

bad=0
while IFS= read -r q; do
  [ -z "$q" ] && continue
  out=$($MYSQL -N -e "$q" 2>/dev/null || true)
  if [ -n "$out" ]; then echo "  [坏] $out"; bad=$((bad+1)); fi
done <<< "$gen"
[ "$bad" -eq 0 ] && echo "  全库无坏数据 ✅"

echo "=== [2/2] 核查字符集是否统一 utf8mb4 ==="
notmb4=$($MYSQL -N -e "SELECT table_name, column_name, character_set_name FROM information_schema.columns WHERE table_schema='$DB' AND character_set_name IS NOT NULL AND character_set_name<>'utf8mb4';" 2>/dev/null || true)
if [ -n "$notmb4" ]; then
  echo "  [警告] 以下文本列非 utf8mb4(以后可能写坏)："; echo "$notmb4"
else
  echo "  库/表/列字符集全部 utf8mb4 ✅"
fi

echo "=== 完成 ==="
