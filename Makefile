	
x86build:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.GitCommit=${shell git rev-list -1 HEAD}" -o blescanner blescanner.go

armbuild:
	GOOS=linux GOARCH=arm go build -ldflags "-X main.GitCommit=${shell git rev-list -1 HEAD}" -o blescanner blescanner.go

x86run: x86build
	sudo ./blescanner

armrun: armbuild
	sudo ./blescanner

x86debug:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.GitCommit=${shell git rev-list -1 HEAD}" -gcflags="all=-N -l" -o blescanner blescanner.go

push: armbuild
	$(shell which balena) push blescanner





