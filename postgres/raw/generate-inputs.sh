#!/bin/bash

# Variáveis globais
INPUT_BOX_ADDRESS=0x593E5BCf894D6829Dd26D0810DA7F064406aebB6
APPLICATION_ADDRESS=0x75135d8ADb7180640d29d822D9AD59E83E8695b2
MNEMONIC="test test test test test test test test test test test junk"
RPC_URL="http://localhost:8545"

# Loop para executar o comando 101 vezes com INPUT sequencial em hexadecimal
for ((i=0; i<101; i++)); do
    # Define o INPUT como hexadecimal sequencial
    INPUT=$(printf "0x%x" $i)

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
