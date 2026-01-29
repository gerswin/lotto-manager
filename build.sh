#!/bin/bash

echo "Compilando servidor..."
go mod tidy
go build -o server ./cmd/server/

if [ $? -eq 0 ]; then
    echo "Compilación exitosa: ./server"
else
    echo "Error en la compilación"
    exit 1
fi
