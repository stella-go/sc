
.PHONY: all sc-linux-amd64 sc-darwin-amd64 sc-darwin-arm64 sc-windows-amd64 clean

all: sc-linux-amd64 sc-darwin-amd64 sc-darwin-arm64 sc-windows-amd64

sc-linux-amd64: releases
	@GOOS=linux GOARCH=amd64 go build -o sc main.go && tar -zcf releases/$@.tar.gz sc && rm sc

sc-darwin-amd64: releases
	@GOOS=darwin GOARCH=amd64 go build -o sc main.go && tar -zcf releases/$@.tar.gz sc && rm sc

sc-darwin-arm64: releases
	@GOOS=darwin GOARCH=arm64 go build -o sc main.go && tar -zcf releases/$@.tar.gz sc && rm sc

sc-windows-amd64: releases
	@GOOS=windows GOARCH=amd64 go build -o sc.exe main.go && zip -qry releases/$@.zip sc && rm sc

releases: clean
	@mkdir -p releases

clean:
	@rm -rf releases