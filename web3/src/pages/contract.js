import * as Web3 from "web3";

var web3 = null;
var networkId ='';
const contractAddrMap = {
    '5': '0xcE25460c82A2dE7D4bBEd1fA98C4a3f27f6362df',
    '1': '0xe603a62a62F024D8323c3b7BcacEbB87d179b61C',
    '421613': '0x7B6E05a55B1756f827F205BF454BF75288904ecF'
}
const contractABI = [
    {
        "inputs": [
            {
                "internalType": "bytes",
                "name": "signature",
                "type": "bytes"
            },
            {
                "internalType": "bytes",
                "name": "message",
                "type": "bytes"
            }
        ],
        "name": "Deposit",
        "outputs": [],
        "stateMutability": "payable",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "address",
                "name": "wallet",
                "type": "address"
            },
            {
                "internalType": "uint256",
                "name": "feeRate",
                "type": "uint256"
            }
        ],
        "stateMutability": "nonpayable",
        "type": "constructor"
    },
    {
        "anonymous": false,
        "inputs": [
            {
                "indexed": true,
                "internalType": "address",
                "name": "previousOwner",
                "type": "address"
            },
            {
                "indexed": true,
                "internalType": "address",
                "name": "newOwner",
                "type": "address"
            }
        ],
        "name": "OwnershipTransferred",
        "type": "event"
    },
    {
        "inputs": [
            {
                "internalType": "bytes",
                "name": "userID",
                "type": "bytes"
            },
            {
                "internalType": "address",
                "name": "wallet",
                "type": "address"
            },
            {
                "internalType": "bytes",
                "name": "signature",
                "type": "bytes"
            },
            {
                "internalType": "bytes",
                "name": "message",
                "type": "bytes"
            }
        ],
        "name": "Rebind",
        "outputs": [],
        "stateMutability": "payable",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "bytes",
                "name": "userID",
                "type": "bytes"
            },
            {
                "internalType": "bytes",
                "name": "recoverID",
                "type": "bytes"
            },
            {
                "internalType": "bytes",
                "name": "web3Key",
                "type": "bytes"
            },
            {
                "internalType": "bytes",
                "name": "web3Public",
                "type": "bytes"
            }
        ],
        "name": "Register",
        "outputs": [],
        "stateMutability": "payable",
        "type": "function"
    },
    {
        "inputs": [],
        "name": "renounceOwnership",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "address",
                "name": "newOwner",
                "type": "address"
            }
        ],
        "name": "transferOwnership",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "bytes",
                "name": "signature",
                "type": "bytes"
            },
            {
                "internalType": "bytes",
                "name": "message",
                "type": "bytes"
            },
            {
                "internalType": "uint256",
                "name": "amount",
                "type": "uint256"
            }
        ],
        "name": "Withdraw",
        "outputs": [
            {
                "internalType": "uint256",
                "name": "",
                "type": "uint256"
            }
        ],
        "stateMutability": "payable",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "bytes",
                "name": "userID",
                "type": "bytes"
            },
            {
                "internalType": "bytes",
                "name": "signature",
                "type": "bytes"
            },
            {
                "internalType": "bytes",
                "name": "message",
                "type": "bytes"
            }
        ],
        "name": "Load",
        "outputs": [
            {
                "internalType": "bytes",
                "name": "recoverID",
                "type": "bytes"
            },
            {
                "internalType": "bytes",
                "name": "web3Public",
                "type": "bytes"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [],
        "name": "Meta",
        "outputs": [
            {
                "internalType": "uint256",
                "name": "feeRate",
                "type": "uint256"
            },
            {
                "internalType": "uint256",
                "name": "registTotal",
                "type": "uint256"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [],
        "name": "owner",
        "outputs": [
            {
                "internalType": "address",
                "name": "",
                "type": "address"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "bytes",
                "name": "userID",
                "type": "bytes"
            }
        ],
        "name": "Web3Key",
        "outputs": [
            {
                "internalType": "bytes",
                "name": "web3Key",
                "type": "bytes"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    }
]
var ShowMsg = function(error, msg){}

/*
example:
    Init(this.ShowMsg, web3Provider);
*/
async function ContractInit(showMsg, provider) {
    ShowMsg = showMsg;

    const vweb3 = new Web3(provider);
    const vnetworkId = await vweb3.eth.net.getId();
    console.log('wallet connect successed: ', vnetworkId, vweb3, provider);
    if (contractAddrMap[vnetworkId] === undefined) {
        ShowMsg('error', 'Unsupport network, currently supported chainId list: ' + Object.keys(contractAddrMap));
        return;
    }
    web3 = vweb3;
    networkId = vnetworkId;
}

function GetWeb3() {
    return web3;
}

function GetNetworkID() {
    return networkId;
}

function ContractReset() {
    web3 = null;
    networkId = '';
}

/*
example:
    var loadParams = [];
    loadParams.push(Web3.utils.asciiToHex(selfID));
    loadParams.push(sig);
    loadParams.push(Web3.utils.asciiToHex(message));
    Execute("call", "Load", self.walletAddress, 0, loadParams, function (loadResult) {
        console.log('web3 contract Load successed: ', loadResult);
    }, function (err) {
        console.log('web3 contract Load failed: ', err);
    });
*/
async function ContractExecute(executeFunc, methodName, walletAddress, msgValue, params, successed, failed) {
    console.log(contractAddrMap[networkId], contractABI, executeFunc, methodName, walletAddress, msgValue, params);
    const myContract = new web3.eth.Contract(contractABI, contractAddrMap[networkId]);
    let web3Func = myContract.methods[methodName];

    let self = this;
    let sendObject = {};
    if (params.length === 0) {
        sendObject = web3Func();
    } else {
        sendObject = web3Func(...params);
    }
    if (msgValue !== undefined && msgValue > 0) msgValue = Web3.utils.toBN(Web3.utils.toWei(msgValue + '', 'ether'));

    if (executeFunc === 'call') {
        await sendObject.call({ from: walletAddress }, function (error, result) {
            if (error) {
                console.log("Execute failed: ", error['message']);
                if (failed !== undefined && failed !== null) failed(error['message']);
            } else {
                console.log("Execute successed: ", result);
                if (successed !== undefined && successed !== null) successed(result);
            }
        })
    } else if (executeFunc === 'send') {
        const gasAmount = await sendObject.estimateGas({ from: walletAddress, value: msgValue });
        console.log('gasLimit', gasAmount);
        await sendObject.send({ from: walletAddress, value: msgValue, gasLimit: gasAmount })
            .on('transactionHash', function (hash) {
                console.log('transactionHash:', hash);
                // self.$Message.success('web3Execute run succesed: ', hash);
                ShowMsg('error', 'web3Execute run succesed: ', hash)
            })
            .on('confirmation', function (confirmationNumber, receipt) {
            })
            .on('receipt', function (receipt) {
                console.log("Execute successed: ", receipt);
                if (successed !== undefined && successed !== null) successed(receipt);
            })
            .on('error', function(error){
                console.log("Execute failed: ", error);
                if (failed !== undefined && failed !== null) failed(error['message']);
            });
    }
}

export {ContractInit, ContractReset, ContractExecute, GetWeb3, GetNetworkID}