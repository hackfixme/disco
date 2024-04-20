version := "1"

# Shell script recipes fail if /tmp is mounted with noexec, so change the
# tempdir. See https://github.com/casey/just/issues/1611
set tempdir := "."


default:
  just --list


build *ARGS:
  ./release/build.sh '{{ARGS}}'


clean:
  rm -rf ./dist ./golangci-lint*.txt


lint report="":
  #!/usr/bin/env sh
  if [ -z '{{report}}' ]; then
    golangci-lint run --out-format=tab --new-from-rev=e72df147cd ./...
    exit $?
  fi

  _report_id="$(date '+%Y%m%d')-$(git describe --tags --abbrev=10 --always)"
  golangci-lint run --out-format=tab --issues-exit-code=0 ./... | \
    tee "golangci-lint-${_report_id}.txt" | \
      awk 'NF {if ($2 == "revive") print $2 ":" $3; else print $2}' \
      | sort | uniq -c | sort -nr \
      | tee "golangci-lint-summary-${_report_id}.txt"


test target="..." *ARGS="":
  go test -v -race -count=1 -failfast {{ARGS}} ./{{target}}