all: server client

out:
	@mkdir -p ./out

.PHONY: server
server: out clean
	@cd ./server && go build -o ../out/server

.PHONY: client
client: out clean
	@cd ./client && go build -o ../out/client

.PHONY: clean
clean:
	@rm ./out/client
	@rm ./out/server
