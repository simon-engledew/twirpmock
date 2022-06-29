FILES := $(shell git ls-files)
FILES += $(shell git ls-files --others --exclude-standard)
FILES := $(filter-out $(shell git ls-files --deleted), $(FILES))

search = $(filter $(1),$(FILES))

docker/root/twirpmock: go.mod go.sum $(call search,%.go)
	GOOS=linux go build -o $@ .

example/service.proto.pb: example/service.proto
	protoc --include_imports --descriptor_set_out=$@ -I . $<

.docker: $(call search,docker/%)
	docker build docker/ --iidfile .docker -t twirpmock

.PHONY: example
example: .docker example/service.proto.pb
	# curl -X POST http://localhost:8888/twirp/twirpmock.example.Example/Echo -H 'Content-Type: application/json' -d '{"name": "There"}'
	docker run --rm -p 8888:8888 -v "$(PWD)/example:/data" twirpmock /data/service.proto.pb /data/service.star
