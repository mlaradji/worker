#!/bin/sh

# CA 1
mkdir -p ca1
openssl req -x509 -newkey rsa:4096 -days 365 -nodes -keyout ca1/key.pem -out ca1/cert.pem -subj "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=CA/CN=Trusted CA/emailAddress=mlaradji@pm.me"

# Server
mkdir -p server
openssl req -newkey rsa:4096 -nodes -keyout server/key.pem -out server/req.pem -subj "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=Server/CN=Server/emailAddress=mlaradji@pm.me"
openssl x509 -CAcreateserial -req -in server/req.pem -CA ca1/cert.pem -CAkey ca1/key.pem -out server/cert.pem -extfile ext.cnf

# Client 1
mkdir -p client1
openssl req -newkey rsa:4096 -nodes -keyout client1/key.pem -out client1/req.pem -subj "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=Client/CN=Client 1/emailAddress=mlaradji@pm.me"
openssl x509 -CAcreateserial -req -in client1/req.pem -days 60 -CA ca1/cert.pem -CAkey ca1/key.pem -out client1/cert.pem -extfile ext.cnf

# Client 2
mkdir -p client2
openssl req -newkey rsa:4096 -nodes -keyout client2/key.pem -out client2/req.pem -subj "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=Client/CN=Client 2/emailAddress=mlaradji@pm.me"
openssl x509 -CAcreateserial -req -in client2/req.pem -days 60 -CA ca1/cert.pem -CAkey ca1/key.pem -out client2/cert.pem -extfile ext.cnf

# Client 3
mkdir -p client3
openssl req -newkey rsa:4096 -nodes -keyout client3/key.pem -out client3/req.pem -subj "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=Client/CN=Client 3/emailAddress=mlaradji@pm.me"
openssl x509 -CAcreateserial -req -in client3/req.pem -days 60 -CA ca1/cert.pem -CAkey ca1/key.pem -out client3/cert.pem -extfile ext.cnf

# CA 2
mkdir -p ca2
openssl req -x509 -newkey rsa:4096 -days 365 -nodes -keyout ca2/key.pem -out ca2/cert.pem -subj "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=CA/CN=Untrusted CA/emailAddress=mlaradji@pm.me"

# Client 4
mkdir -p client4
openssl req -newkey rsa:4096 -nodes -keyout client4/key.pem -out client4/req.pem -subj "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=Client/CN=Client 3/emailAddress=mlaradji@pm.me"
openssl x509 -CAcreateserial -req -in client4/req.pem -days 60 -CA ca2/cert.pem -CAkey ca2/key.pem -out client4/cert.pem -extfile ext.cnf
