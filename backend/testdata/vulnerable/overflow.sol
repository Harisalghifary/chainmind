// SPDX-License-Identifier: MIT
pragma solidity ^0.7.6;

contract OverflowToken {
    mapping(address => uint256) public balances;
    uint256 public totalSupply;

    function mint(uint256 amount) external {
        totalSupply += amount;
        balances[msg.sender] += amount;
    }

    function batchTransfer(address[] calldata recipients, uint256 amount) external {
        for (uint256 i = 0; i < recipients.length; i++) {
            balances[msg.sender] -= amount;
            balances[recipients[i]] += amount;
        }
    }
}