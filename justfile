default:
  just --list

version := "1"

build *ARGS:
  ./release/build.sh "{{ARGS}}"

clean:
  rm -rf ./dist