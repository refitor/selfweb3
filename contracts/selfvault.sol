//SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/access/Ownable.sol";
import '@openzeppelin/contracts/utils/Address.sol';
import '@openzeppelin/contracts/utils/math/SafeMath.sol';

import "./SelfWeb3.sol";

// @title The SelfVault contract is used to provide web3 fund protection services.
// @author refitor
contract SelfVault is Ownable, SelfWeb3 {
    using SafeMath for uint256;
    using Address for address payable;
    struct VaultMeta {
        address wallet;
        uint256 feeRate;
    }
    VaultMeta private _vaultMeta;
    mapping (address => uint256) private _vaultMap;

    /**
     * @dev constructor is used to populate the meta information of the contract.
     * @param wallet the wallet address specified by the creator.
     * @param web2Address The web2 server address.
     * @param feeRate fee deduction percentage set by the contract creator, demo: 500 / 10000, feeRate is 500.
     */
    constructor(address wallet, address web2Address, uint256 feeRate) SelfWeb3(web2Address) {
        _vaultMeta = VaultMeta(wallet, 0);
        _vaultMeta.feeRate = feeRate;
    }

    /**
     * @dev Deposit is used to deposit native assets supported by selfWeb3.
     * @param selfAddress The user’s unique address in selfWeb3.
     * @param vparam Used for on-chain verification.
     */
    function Deposit(address selfAddress, bytes memory vparam) external payable {
        MetaData memory md = _get();
        SelfData memory sd = _getKV(selfAddress);

        // on-chain associated validation
        address[] memory sigAddrList = new address[](2);
        sigAddrList[0] = selfAddress;
        sigAddrList[1] = md.Web2Address;
        bytes32 verifyRoot = SelfValidator.RelateVerify(sd.VerifyRoot, vparam, sigAddrList); // Prioritize verification
        if(verifyRoot != sd.VerifyRoot && verifyRoot != bytes32(0)) {
            sd.VerifyRoot = verifyRoot;
            _setKV(selfAddress, sd);
        }
        delete sigAddrList;

        require(msg.value > 0, "invalid deposited amount");
        require(sd.SelfPrivate.length != 0, "not registered yet");

        // on-chain vault management
        _setVault(address(0), _getVault(address(0)) + msg.value);
    }

    /**
     * @dev Withdraw is used to withdraw native assets supported by selfWeb3.
     * @param selfAddress The user’s unique address in selfWeb3.
     * @param vparam Used for on-chain verification.
     * @param amount withdrawal native assets amount.
     */
    function Withdraw(address selfAddress, bytes memory vparam, uint256 amount) external payable returns (uint256) {
        MetaData memory md = _get();
        SelfData memory sd = _getKV(selfAddress);

        // on-chain associated validation
         address[] memory sigAddrList = new address[](2);
        sigAddrList[0] = selfAddress;
        sigAddrList[1] = md.Web2Address;
        bytes32 verifyRoot = SelfValidator.RelateVerify(sd.VerifyRoot, vparam, sigAddrList); // Prioritize verification
        if(verifyRoot != sd.VerifyRoot && verifyRoot != bytes32(0)) {
            sd.VerifyRoot = verifyRoot;
            _setKV(selfAddress, sd);
        }
        delete sigAddrList;

        require(amount > 0, "invalid withdraw amount");
        require(sd.SelfPrivate.length != 0, "not registered yet");

        // on-chain vault management
        VaultMeta memory vm = _getVaultMeta();
        if (vm.feeRate > 0) {
            uint256 payFee = amount * vm.feeRate / 10000;
            require(amount - payFee > 0 && amount - payFee <= _getVault(address(0)), "not enough extractable quantity");
            payable(vm.wallet).sendValue(payFee);
            amount = amount - payFee;
        } else {
            require(amount > 0 && amount <= _getVault(address(0)), "not enough extractable quantity");
        }
        _setVault(address(0), _getVault(address(0)) - amount);
        return _getVault(address(0));
    }

    /**
     * @dev FeeRate is used to load the feeRate information of the contract.
     */
    function FeeRate() view external returns (uint256 feeRate) {
        VaultMeta memory md = _getVaultMeta();
        return (md.feeRate);
    }

    /**
     * @dev _setKV is used to set dataMap.
     * @param k the key for _datamap.
     * @param v the value for _datamap.
     */
    function _setVault(address k, uint256 v) private {
        _vaultMap[k] = v;
    }

    /**
     * @dev _getKV is used to get the value by the key.
     * @param k the key for _datamap.
     */
    function _getVault(address k) private view returns (uint256) {
        return _vaultMap[k];
    }

    /**
     * @dev _get is used to load the meta information of the contract.
     */
    function _getVaultMeta() private view returns (VaultMeta memory md) {
        return _vaultMeta;
    }
}