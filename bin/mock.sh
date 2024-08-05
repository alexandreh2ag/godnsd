#!/bin/bash

go install go.uber.org/mock/mockgen@latest
rm -rf mocks
mockgen -destination=mocks/docker/docker.go -package=mockDocker github.com/docker/docker/client APIClient
mockgen -destination=mocks/types/mock.go -package=mockTypes github.com/alexandreh2ag/go-dns-discover/types Provider
mockgen -destination=mocks/miekg/dns.go -package=mockMiekgDns github.com/miekg/dns ResponseWriter
