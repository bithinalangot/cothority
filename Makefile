all: test_fmt test_lint test_local

# gopkg fits all v1.1, v1.2, ... in v1
PKG_STABLE = gopkg.in/dedis/onchain-secrets.v1
include $(GOPATH)/src/github.com/dedis/Coding/bin/Makefile.base
EXCLUDE_LINT = "should be.*UI|_test.go"

# You can use `test_playground` to run any test or part of cothority
# for more than once in Travis. Change `make test` in .travis.yml
# to `make test_playground`.
test_playground:
	cd skipchain; \
	for a in $$( seq 100 ); do \
	  go test -v -race -short || exit 1 ; \
	done;

# Other targets are:
# make create_stable

IMAGE_NAME = dedis/onchain-secrets

docker:
	docker build -t $(IMAGE_NAME) .

docker_run:
	docker run -it --rm -p 7003:7003 -p 7005:7005 -p 7007:7007 --name ocs \
	 -v $(pwd)/data:/root/.local/share/conode $(IMAGE_NAME)

proto:
	awk -f proto.awk struct.go > external/proto/ocs.proto
	awk -f proto.awk darc/struct.go > external/proto/darc.proto
	cd external; make
