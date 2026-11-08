#!/usr/bin/env bash
# Migration immutability gate (the Miuno check-migrations.sh shape, adapted
# to migrations embedded in internal/db/db.go). Published migration versions
# live at HEAD: an already-published version must never be edited, deleted,
# or renumbered, and a new version must be strictly greater than the
# published maximum. Run in CI alongside vet/test.
set -euo pipefail

cd "$(dirname "$0")/.."

file=internal/db/db.go
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

extract_versions() {
    # Every "N: `" entry, in numeric order.
    grep -oE '^[[:space:]]+[0-9]+: `' "$1" | grep -oE '[0-9]+' | sort -n
}

extract_block() {
    # Print the raw-string body of one migration: from its "N: `" line up to
    # and including the closing backtick-comma line. SQL never contains a
    # line that is exactly "`," so the boundary is unambiguous.
    awk -v start="^[[:space:]]+$2: \`[[:space:]]*$" '
        found && /^`,[[:space:]]*$/ { print; exit }
        $0 ~ start { found = 1 }
        found { print }
    ' "$1"
}

# Nothing to check when db.go itself is untouched.
if git diff --quiet HEAD -- "$file" && ! git diff --cached --quiet HEAD -- "$file" 2>/dev/null; then
    :
fi
if git diff HEAD -- "$file" | head -1 | grep -q '^diff'; then
    :
else
    # Unchanged against HEAD (covers staged-only comparisons too).
    if git diff --quiet HEAD -- "$file"; then
        exit 0
    fi
fi

git show HEAD:"$file" > "$tmp/head.go" 2>/dev/null || {
    echo "check-migrations: $file is new; nothing published to protect" >&2
    exit 0
}

mapfile -t head_versions < <(extract_versions "$tmp/head.go")
mapfile -t wt_versions < <(extract_versions "$file")

if [ "${#head_versions[@]}" -eq 0 ]; then
    echo "check-migrations: no migration entries found at HEAD in $file" >&2
    exit 1
fi

head_max=${head_versions[${#head_versions[@]}-1]}
fail=0

for v in "${head_versions[@]}"; do
    extract_block "$tmp/head.go" "$v" > "$tmp/head-$v.sql"
    extract_block "$file" "$v" > "$tmp/wt-$v.sql"
    if [ ! -s "$tmp/wt-$v.sql" ]; then
        echo "FAIL: migration $v was deleted or renumbered (applied versions are immutable)" >&2
        fail=1
        continue
    fi
    if ! cmp -s "$tmp/head-$v.sql" "$tmp/wt-$v.sql"; then
        echo "FAIL: migration $v was edited (applied versions are immutable; add version $((head_max + 1)) instead)" >&2
        diff "$tmp/head-$v.sql" "$tmp/wt-$v.sql" | head >&2
        fail=1
    fi
done

for v in "${wt_versions[@]}"; do
    known=0
    for h in "${head_versions[@]}"; do
        [ "$v" = "$h" ] && { known=1; break; }
    done
    if [ "$known" -eq 0 ] && [ "$v" -le "$head_max" ]; then
        echo "FAIL: new migration $v is not greater than published max $head_max (versions must be strictly increasing)" >&2
        fail=1
    fi
done

# Guard against duplicate keys sneaking past the map literal.
dups=$(printf '%s\n' "${wt_versions[@]}" | uniq -d)
if [ -n "$dups" ]; then
    echo "FAIL: duplicate migration version entries: $dups" >&2
    fail=1
fi

exit "$fail"
