#!/bin/bash

if [ ! -f "./server" ]; then
    echo "Servidor no encontrado. Compilando..."
    ./build.sh
fi

echo "Iniciando servidor..."
./server
