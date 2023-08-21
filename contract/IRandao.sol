// SPDX-License-Identifier: GPL-2.0
pragma solidity ^0.8.9;

// version 1.0
interface IRandao {
    event LogCampaignAdded(
        uint256 indexed campaignID,
        address indexed from,
        uint256 currbNum,
        uint256 indexed bnum,
        uint256 deposit,
        uint16 commitBalkline,
        uint16 commitDeadline,
        uint256 bountypot,
        uint256 maxParticipant
    );

    event LogFollow(
        uint256 indexed CampaignId,
        address indexed from,
        uint256 bountypot
    );

    event LogCommit(
        uint256 indexed CampaignId,
        address indexed from,
        bytes32 commitment
    );
    event LogReveal(
        uint256 indexed CampaignId,
        address indexed from,
        uint256 secret
    );

    event LogGetRandom(uint256 indexed CampaignId, uint256 indexed random);

    struct Participant {
        uint256 secret;
        bytes32 commitment;
        uint256 reward;
        bool revealed;
        bool rewarded;
    }

    struct Consumer {
        address caddr;
        uint256 bountypot;
    }

    struct CampaignInfo {
        uint256 bnum;
        uint256 deposit;
        uint16 commitBalkline;
        uint16 commitDeadline;
        uint256 random;
        bool settled;
        uint256 bountypot;
        uint32 commitNum;
        uint32 revealsNum;
    }

    struct Campaign {
        uint256 bnum;
        uint256 deposit;
        uint16 commitBalkline;
        uint16 commitDeadline;
        uint256 random;
        bool settled;
        uint256 bountypot;
        uint32 commitNum;
        uint32 revealsNum;
        uint256 maxParticipant;
        mapping(address => Consumer) consumers;
        mapping(address => Participant) participants;
        mapping(bytes32 => bool) commitments;
    }

    function newCampaign(
        uint256 _bnum,
        uint256 _deposit,
        uint16 _commitBalkline,
        uint16 _commitDeadline,
        uint256 _maxTxFee
    ) external payable returns (uint256 _campaignID);

    function getCampaign(
        uint256 _campaignID
    ) external view returns (CampaignInfo memory);

    function follow(uint256 _campaignID) external payable returns (bool);

    function commit(uint256 _campaignID, bytes32 _hs) external payable;

    // For test
    function getCommitment(uint256 _campaignID) external view returns (bytes32);

    function shaCommit(uint256 _s) external pure returns (bytes32);

    function reveal(uint256 _campaignID, uint256 _s) external;

    function getRandom(uint256 _campaignID) external returns (uint256);

    function getMyBounty(uint256 _campaignID) external returns (uint256);

    function refundBounty(uint256 _campaignID) external;

    function numCampaigns() external view returns (uint256);
}
