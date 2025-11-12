all:
	go run -v ./make.go build

clean:
	rm -rf ./build/
	rm -f output/velociraptor*
