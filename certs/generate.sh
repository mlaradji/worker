#!/bin/sh

# CA
mkdir -p ca
openssl req -x509 -newkey rsa:4096 -days 365 -nodes -keyout ca/key.pem -out ca/cert.pem -subj "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=CA/CN=CA/emailAddress=mlaradji@pm.me"

# Server
mkdir -p server
openssl req -newkey rsa:4096 -nodes -keyout server/key.pem -out server/req.pem -subj "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=Server/CN=Server/emailAddress=mlaradji@pm.me"
openssl x509 -CAcreateserial -req -in server/req.pem -CA ca/cert.pem -CAkey ca/key.pem -out server/cert.pem -extfile ext.cnf

# Client 1
mkdir -p client1
openssl req -newkey rsa:4096 -nodes -keyout client1/key.pem -out client1/req.pem -subj "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=Client/CN=Client 1/emailAddress=mlaradji@pm.me"
openssl x509 -CAcreateserial -req -in client1/req.pem -days 60 -CA ca/cert.pem -CAkey ca/key.pem -out client1/cert.pem -extfile ext.cnf

# Client 2
mkdir -p client2
openssl req -newkey rsa:4096 -nodes -keyout client2/key.pem -out client2/req.pem -subj "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=Client/CN=Client 2/emailAddress=mlaradji@pm.me"
openssl x509 -CAcreateserial -req -in client2/req.pem -days 60 -CA ca/cert.pem -CAkey ca/key.pem -out client2/cert.pem -extfile ext.cnf

# Client 3
mkdir -p client3
openssl req -newkey rsa:4096 -nodes -keyout client3/key.pem -out client3/req.pem -subj "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=Client/CN=Client 3/emailAddress=mlaradji@pm.me"
openssl x509 -CAcreateserial -req -in client3/req.pem -days 60 -CA ca/cert.pem -CAkey ca/key.pem -out client3/cert.pem -extfile ext.cnf