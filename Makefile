.PHONY: skills clean test coverage

skills:
	cd skills/build_skill    && go build -o build  build.go
	cd skills/file_search    && go build -o search search.go
	cd skills/grep           && go build -o grep   grep.go
	cd skills/list_directory && go build -o list   list.go
	cd skills/read_file      && go build -o read   read.go
	cd skills/wiki_search    && go build -o search search.go
	cd skills/write_file     && go build -o write  write.go

test:
	go test ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean:
	rm -f skills/build_skill/build
	rm -f skills/file_search/search
	rm -f skills/grep/grep
	rm -f skills/list_directory/list
	rm -f skills/read_file/read
	rm -f skills/wiki_search/search
	rm -f skills/write_file/write
