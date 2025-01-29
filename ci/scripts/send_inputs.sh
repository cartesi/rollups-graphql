#!/bin/bash

# Global variables
INPUT_BOX_ADDRESS=0x593E5BCf894D6829Dd26D0810DA7F064406aebB6
APPLICATION_ADDRESS=0x36B9E60ACb181da458aa8870646395CD27cD0E6E
MNEMONIC="test test test test test test test test test test test junk"
RPC_URL="http://localhost:8545"

# Loop for execute the command 101 times with sequential hexadecimal INPUT
for ((i=0; i<101; i++)); do
    # Define the INPUT like a hexadecimal with 2 digits (padding with zero on the left)
    INPUT=$(printf "0xdeadbeef%02x" $i)

    # Execute the command in line
    echo "Running the command $((i+1)) of 101 with INPUT=$INPUT..."
    cast send \
        --mnemonic "$MNEMONIC" \
        --rpc-url "$RPC_URL" \
        $INPUT_BOX_ADDRESS "addInput(address,bytes)(bytes32)" $APPLICATION_ADDRESS $INPUT

    # Verify if the last command was successful
    if [[ $? -ne 0 ]]; then
        echo "Error executing the command $((i+1)). Stopping the script."
        exit 1
    fi
done

echo "All commands were executed successfully!"