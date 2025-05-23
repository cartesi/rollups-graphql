#!/bin/bash

# Global variables
INPUT_BOX_ADDRESS=0xB6b39Fb3dD926A9e3FBc7A129540eEbeA3016a6c
APPLICATION_ADDRESS=0x8e3c7bF65833ccb1755dAB530Ef0405644FE6ae3
PRIVATE_KEY="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
RPC_URL="http://localhost:8545"

# Loop for execute the command 101 times with sequential hexadecimal INPUT
for ((i=0; i<101; i++)); do
    # Define the INPUT like a hexadecimal with 2 digits (padding with zero on the left)
    INPUT=$(printf "0xdeadbeef%02x" $i)

    # Execute the command in line
    echo "Running the command $((i+1)) of 101 with INPUT=$INPUT..."
    cast send \
        --private-key "$PRIVATE_KEY" \
        --rpc-url "$RPC_URL" \
        --nonce $i\
        --async \
        $INPUT_BOX_ADDRESS "addInput(address,bytes)(bytes32)" $APPLICATION_ADDRESS $INPUT

    # Verify if the last command was successful
    if [[ $? -ne 0 ]]; then
        echo "Error executing the command $((i+1)). Stopping the script."
        exit 1
    fi
done
sleep 10
echo "All commands were executed successfully!"