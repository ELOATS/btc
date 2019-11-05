#!/bin/bash
rm -f *.db
rm -f blockchain

go build -o blockchain *.go
./blockchain
