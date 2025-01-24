#!/bin/bash

# Variáveis globais
INPUT_BOX_ADDRESS=0x593E5BCf894D6829Dd26D0810DA7F064406aebB6
APPLICATION_ADDRESS=0x36B9E60ACb181da458aa8870646395CD27cD0E6E
MNEMONIC="test test test test test test test test test test test junk"
RPC_URL="http://localhost:8545"

# Loop para executar o comando 101 vezes com INPUT sequencial em hexadecimal
for ((i=0; i<101; i++)); do
    # Define o INPUT como hexadecimal sequencial com 2 dígitos (preenchendo com zero à esquerda)
    INPUT=$(printf "0xdeadbeef%02x" $i)

    # Executa o comando em linha
    echo "Executando comando $((i+1)) de 101 com INPUT=$INPUT..."
    cast send \
        --mnemonic "$MNEMONIC" \
        --rpc-url "$RPC_URL" \
        $INPUT_BOX_ADDRESS "addInput(address,bytes)(bytes32)" $APPLICATION_ADDRESS $INPUT

    # Verifica se o último comando foi bem-sucedido
    if [[ $? -ne 0 ]]; then
        echo "Erro ao executar o comando $((i+1)). Interrompendo o script."
        exit 1
    fi
done

echo "Todos os comandos foram executados com sucesso!"
