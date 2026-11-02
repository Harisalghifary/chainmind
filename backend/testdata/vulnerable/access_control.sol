// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract UnprotectedAdmin {
    address public owner;
    uint256 public totalSupply;

    constructor() {
        owner = msg.sender;
    }

    function mint(uint256 amount) external {
        totalSupply += amount;
    }

    function destroy() external {
        selfdestruct(payable(msg.sender));
    }
}