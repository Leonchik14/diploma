#!/bin/bash

openssl genrsa -out jwt_private.pem 2048
openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem

echo "Keys generated:"
echo "Private key: jwt_private.pem"
echo "Public key: jwt_public.pem"
echo ""
echo "Set environment variables:"
echo "export JWT_PRIVATE_KEY=\"\$(cat jwt_private.pem)\""
echo "export JWT_PUBLIC_KEY=\"\$(cat jwt_public.pem)\""
