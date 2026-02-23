#!/bin/bash
#
# Claude Code Hook: feat commit 시 docs 업데이트 여부 검사
#
# 기능 추가(feat) commit에서 소스 코드가 변경되었는데
# docs/ 디렉토리에 변경사항이 없으면 commit을 차단합니다.
# fix, refactor, test, chore 등 기능 추가가 아닌 commit은 통과합니다.
#

# stdin에서 hook input JSON 읽기
INPUT=$(cat)

# Bash 명령어 추출
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // ""')

# git commit 명령이 아니면 통과
if ! echo "$COMMAND" | grep -qE 'git\s+commit'; then
  exit 0
fi

# commit 메시지가 feat으로 시작하는지 확인
# Claude Code의 commit 명령 형식:
#   git commit -m "feat: ..."
#   git commit -m "$(cat <<'EOF'\nfeat: ...\nEOF\n)"
# 명령 전체에서 feat 키워드 존재 여부로 판단
IS_FEAT=false
if echo "$COMMAND" | grep -qiE '(^|[\s"'"'"'(])feat[\s(:]'; then
  IS_FEAT=true
fi

# feat이 아니면 통과 (fix, refactor, test, chore, docs, ci 등)
if [ "$IS_FEAT" = false ]; then
  exit 0
fi

# staged 파일 목록 가져오기
STAGED_FILES=$(git diff --cached --name-only 2>/dev/null || true)

# staged 파일이 없으면 통과 (commit이 실패할 것이므로)
if [ -z "$STAGED_FILES" ]; then
  exit 0
fi

# 소스 코드 변경 감지 패턴
# Go 소스, CRD YAML, config/samples, Makefile 등
SOURCE_PATTERNS="\.go$|config/crd/|config/samples/|config/rbac/|Makefile|\.yaml$|\.yml$"

# docs 변경 감지 패턴
DOCS_PATTERN="^docs/"

# 소스 코드 변경 여부 확인
HAS_SOURCE_CHANGES=false
while IFS= read -r file; do
  if echo "$file" | grep -qE "$SOURCE_PATTERNS"; then
    # docs/ 내부 파일은 제외
    if ! echo "$file" | grep -qE "$DOCS_PATTERN"; then
      HAS_SOURCE_CHANGES=true
      break
    fi
  fi
done <<< "$STAGED_FILES"

# 소스 코드 변경이 없으면 통과 (docs-only 변경, README 수정 등)
if [ "$HAS_SOURCE_CHANGES" = false ]; then
  exit 0
fi

# docs 변경 여부 확인
HAS_DOCS_CHANGES=false
while IFS= read -r file; do
  if echo "$file" | grep -qE "$DOCS_PATTERN"; then
    HAS_DOCS_CHANGES=true
    break
  fi
done <<< "$STAGED_FILES"

# 소스 변경이 있는데 docs 변경이 없으면 차단
if [ "$HAS_DOCS_CHANGES" = false ]; then
  # 변경된 소스 파일 목록 생성
  CHANGED_SOURCE_FILES=$(echo "$STAGED_FILES" | grep -E "$SOURCE_PATTERNS" | grep -vE "$DOCS_PATTERN" || true)

  echo "feat commit: 소스 코드가 변경되었지만 docs/ 디렉토리에 변경사항이 없습니다." >&2
  echo "" >&2
  echo "변경된 소스 파일:" >&2
  while IFS= read -r f; do
    [ -n "$f" ] && echo "  - $f" >&2
  done <<< "$CHANGED_SOURCE_FILES"
  echo "" >&2
  echo "docs/ 디렉토리의 문서를 업데이트한 후 다시 commit 해주세요." >&2
  echo "Docusaurus 문서 위치: docs/content/" >&2
  exit 2
fi

exit 0
