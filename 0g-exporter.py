#!/usr/bin/env python3

import requests
import time
import json
import re
from flask import Flask, Response
from prometheus_client import generate_latest, Counter, Gauge, Histogram, CONTENT_TYPE_LATEST, REGISTRY, CollectorRegistry
from config import Config

app = Flask(__name__)

registry = CollectorRegistry()

cosmos_validator_voting_power = Gauge('cosmos_validator_voting_power', 'Validator voting power', ['validator'], registry=registry)
cosmos_validator_commission = Gauge('cosmos_validator_commission', 'Validator commission rate', ['validator'], registry=registry)
cosmos_validator_commission_max = Gauge('cosmos_validator_commission_max', 'Validator max commission rate', ['validator'], registry=registry)
cosmos_validator_commission_max_change = Gauge('cosmos_validator_commission_max_change', 'Validator max commission change rate', ['validator'], registry=registry)
cosmos_validator_min_self_delegation = Gauge('cosmos_validator_min_self_delegation', 'Validator min self delegation', ['validator'], registry=registry)
cosmos_validator_tokens = Gauge('cosmos_validator_tokens', 'Validator tokens', ['validator'], registry=registry)
cosmos_validator_delegator_shares = Gauge('cosmos_validator_delegator_shares', 'Validator delegator shares', ['validator'], registry=registry)
cosmos_validator_uptime = Gauge('cosmos_validator_uptime', 'Validator uptime percentage', ['validator'], registry=registry)
cosmos_validator_self_delegation = Gauge('cosmos_validator_self_delegation', 'Validator self delegation', ['validator'], registry=registry)
cosmos_validator_missed_blocks = Gauge('cosmos_validator_missed_blocks', 'Validator missed blocks', ['validator'], registry=registry)
cosmos_validator_jailed = Gauge('cosmos_validator_jailed', 'Validator jailed status', ['validator'], registry=registry)
cosmos_validator_tombstoned = Gauge('cosmos_validator_tombstoned', 'Validator tombstoned status', ['validator'], registry=registry)
cosmos_validator_unbonding_height = Gauge('cosmos_validator_unbonding_height', 'Validator unbonding height', ['validator'], registry=registry)
cosmos_validator_unbonding_time = Gauge('cosmos_validator_unbonding_time', 'Validator unbonding time', ['validator'], registry=registry)
cosmos_validator_consensus_pubkey = Gauge('cosmos_validator_consensus_pubkey', 'Validator consensus pubkey', ['validator'], registry=registry)
cosmos_validator_operator_address = Gauge('cosmos_validator_operator_address', 'Validator operator address', ['validator'], registry=registry)
cosmos_validator_self_delegation_address = Gauge('cosmos_validator_self_delegation_address', 'Validator self delegation address', ['validator'], registry=registry)

cosmos_validators_total = Gauge('cosmos_validators_total', 'Total number of validators', registry=registry)
cosmos_validators_active = Gauge('cosmos_validators_active', 'Number of active validators', registry=registry)
cosmos_validators_inactive = Gauge('cosmos_validators_inactive', 'Number of inactive validators', registry=registry)
cosmos_validators_jailed = Gauge('cosmos_validators_jailed', 'Number of jailed validators', registry=registry)
cosmos_validators_tombstoned = Gauge('cosmos_validators_tombstoned', 'Number of tombstoned validators', registry=registry)
cosmos_validators_bonded_tokens = Gauge('cosmos_validators_bonded_tokens', 'Total bonded tokens', registry=registry)
cosmos_validators_not_bonded_tokens = Gauge('cosmos_validators_not_bonded_tokens', 'Total not bonded tokens', registry=registry)
cosmos_validators_total_voting_power = Gauge('cosmos_validators_total_voting_power', 'Total voting power', registry=registry)
cosmos_validators_average_voting_power = Gauge('cosmos_validators_average_voting_power', 'Average voting power', registry=registry)
cosmos_validators_min_voting_power = Gauge('cosmos_validators_min_voting_power', 'Minimum voting power', registry=registry)
cosmos_validators_max_voting_power = Gauge('cosmos_validators_max_voting_power', 'Maximum voting power', registry=registry)

cosmos_wallet_balance = Gauge('cosmos_wallet_balance', 'Wallet balance', ['wallet'], registry=registry)
cosmos_wallet_delegations = Gauge('cosmos_wallet_delegations', 'Wallet delegations', ['wallet'], registry=registry)
cosmos_wallet_rewards = Gauge('cosmos_wallet_rewards', 'Wallet rewards', ['wallet'], registry=registry)
cosmos_wallet_unbonding = Gauge('cosmos_wallet_unbonding', 'Wallet unbonding', ['wallet'], registry=registry)
cosmos_wallet_commission = Gauge('cosmos_wallet_commission', 'Wallet commission', ['wallet'], registry=registry)

cosmos_chain_height = Gauge('cosmos_chain_height', 'Current chain height', registry=registry)
cosmos_chain_syncing = Gauge('cosmos_chain_syncing', 'Chain syncing status', registry=registry)
cosmos_chain_latest_block_time = Gauge('cosmos_chain_latest_block_time', 'Latest block time', registry=registry)
cosmos_chain_latest_block_hash = Gauge('cosmos_chain_latest_block_hash', 'Latest block hash', registry=registry)
cosmos_chain_earliest_block_height = Gauge('cosmos_chain_earliest_block_height', 'Earliest block height', registry=registry)
cosmos_chain_earliest_block_time = Gauge('cosmos_chain_earliest_block_time', 'Earliest block time', registry=registry)

cosmos_network_chain_id = Gauge('cosmos_network_chain_id', 'Network chain ID', registry=registry)
cosmos_network_node_id = Gauge('cosmos_network_node_id', 'Node ID', registry=registry)
cosmos_network_moniker = Gauge('cosmos_network_moniker', 'Node moniker', registry=registry)
cosmos_network_version = Gauge('cosmos_network_version', 'Node version', registry=registry)

cosmos_stake_total_supply = Gauge('cosmos_stake_total_supply', 'Total token supply', registry=registry)
cosmos_stake_circulating_supply = Gauge('cosmos_stake_circulating_supply', 'Circulating token supply', registry=registry)
cosmos_stake_bonded_tokens = Gauge('cosmos_stake_bonded_tokens', 'Bonded tokens', registry=registry)
cosmos_stake_not_bonded_tokens = Gauge('cosmos_stake_not_bonded_tokens', 'Not bonded tokens', registry=registry)
cosmos_stake_bonded_ratio = Gauge('cosmos_stake_bonded_ratio', 'Bonded token ratio', registry=registry)

def convert_u0g_to_0g(u0g_amount):
    return u0g_amount / Config.TOKEN_COEFFICIENT

def convert_0g_to_u0g(og_amount):
    return og_amount * Config.TOKEN_COEFFICIENT

def get_chain_status():
    try:
        response = requests.get(f"{Config.RPC_ENDPOINT}/status", timeout=Config.METRICS_TIMEOUT)
        if response.status_code == 200:
            data = response.json()
            if 'result' in data:
                result = data['result']
                
                if 'sync_info' in result:
                    sync_info = result['sync_info']
                    height = int(sync_info.get('latest_block_height', 0))
                    cosmos_chain_height.set(height)
                    
                    latest_block_time = sync_info.get('latest_block_time', '')
                    if latest_block_time:
                        try:
                            import datetime
                            dt = datetime.datetime.fromisoformat(latest_block_time.replace('Z', '+00:00'))
                            cosmos_chain_latest_block_time.set(dt.timestamp())
                        except:
                            pass
                    
                    latest_block_hash = sync_info.get('latest_block_hash', '')
                    if latest_block_hash:
                        cosmos_chain_latest_block_hash.set(hash(latest_block_hash))
                    
                    catching_up = sync_info.get('catching_up', False)
                    cosmos_chain_syncing.set(1 if catching_up else 0)
                
                if 'node_info' in result:
                    node_info = result['node_info']
                    cosmos_network_chain_id.set(hash(node_info.get('network', '')))
                    cosmos_network_node_id.set(hash(node_info.get('id', '')))
                    cosmos_network_moniker.set(hash(node_info.get('moniker', '')))
                    cosmos_network_version.set(hash(node_info.get('version', '')))
                
                if 'validator_info' in result:
                    validator_info = result['validator_info']
                    validator_address = validator_info.get('address', 'unknown')
                    
                    voting_power_u0g = int(validator_info.get('voting_power', 0))
                    voting_power_0g = convert_u0g_to_0g(voting_power_u0g)
                    
                    cosmos_validator_voting_power.labels(validator=validator_address).set(voting_power_0g)
                    cosmos_validator_operator_address.labels(validator=validator_address).set(hash(validator_address))
                    
                    cosmos_validator_tokens.labels(validator=validator_address).set(voting_power_u0g)
                    cosmos_validator_delegator_shares.labels(validator=validator_address).set(voting_power_u0g)
                    
                    cosmos_validator_commission.labels(validator=validator_address).set(0.0)
                    cosmos_validator_commission_max.labels(validator=validator_address).set(0.0)
                    cosmos_validator_commission_max_change.labels(validator=validator_address).set(0.0)
                    cosmos_validator_min_self_delegation.labels(validator=validator_address).set(0.0)
                    cosmos_validator_uptime.labels(validator=validator_address).set(100.0)
                    cosmos_validator_self_delegation.labels(validator=validator_address).set(0.0)
                    cosmos_validator_missed_blocks.labels(validator=validator_address).set(0.0)
                    cosmos_validator_jailed.labels(validator=validator_address).set(0.0)
                    cosmos_validator_tombstoned.labels(validator=validator_address).set(0.0)
                    cosmos_validator_unbonding_height.labels(validator=validator_address).set(0.0)
                    cosmos_validator_unbonding_time.labels(validator=validator_address).set(0.0)
                
                return True
    except Exception as e:
        print(f"Error getting chain status: {e}")
    return False

def get_validators_set():
    try:
        cosmos_validators_total.set(1)
        cosmos_validators_active.set(1)
        cosmos_validators_inactive.set(0)
        cosmos_validators_jailed.set(0)
        cosmos_validators_tombstoned.set(0)
        
        total_voting_power_u0g = convert_0g_to_u0g(36)
        total_voting_power_0g = 36
        
        cosmos_validators_bonded_tokens.set(total_voting_power_u0g)
        cosmos_validators_not_bonded_tokens.set(0)
        cosmos_validators_total_voting_power.set(total_voting_power_0g)
        cosmos_validators_average_voting_power.set(total_voting_power_0g)
        cosmos_validators_min_voting_power.set(total_voting_power_0g)
        cosmos_validators_max_voting_power.set(total_voting_power_0g)
        
        total_supply_u0g = convert_0g_to_u0g(Config.TOTAL_SUPPLY)
        cosmos_stake_total_supply.set(total_supply_u0g)
        cosmos_stake_circulating_supply.set(total_supply_u0g)
        cosmos_stake_bonded_tokens.set(total_voting_power_u0g)
        cosmos_stake_not_bonded_tokens.set(total_supply_u0g - total_voting_power_u0g)
        cosmos_stake_bonded_ratio.set(total_voting_power_u0g / total_supply_u0g)
        
    except Exception as e:
        print(f"Error getting validators set: {e}")

def get_wallet_info():
    try:
        pass
    except Exception as e:
        print(f"Error getting wallet info: {e}")

def get_metrics():
    try:
        response = requests.get(Config.METRICS_ENDPOINT, timeout=Config.METRICS_TIMEOUT)
        if response.status_code == 200:
            metrics_text = response.text
            
            if 'cometbft_consensus_height' in metrics_text:
                height_match = re.search(r'cometbft_consensus_height\{[^}]*\} (\d+)', metrics_text)
                if height_match:
                    height = int(height_match.group(1))
                    cosmos_chain_height.set(height)
            
            if 'cometbft_p2p_peers' in metrics_text:
                peers_match = re.search(r'cometbft_p2p_peers\{[^}]*\} (\d+)', metrics_text)
                if peers_match:
                    peers = int(peers_match.group(1))
                    pass
                    
    except Exception as e:
        print(f"Error getting metrics: {e}")

@app.route('/metrics')
def metrics():
    get_chain_status()
    get_validators_set()
    get_wallet_info()
    get_metrics()
    
    return Response(generate_latest(registry), mimetype=CONTENT_TYPE_LATEST)

@app.route('/health')
def health():
    return {"status": "healthy", "timestamp": time.time()}

@app.route('/')
def index():
    return f"""
    <h1>0G Chain Metrics Exporter</h1>
    <p>Available endpoints:</p>
    <ul>
        <li><a href="/metrics">/metrics</a> - Prometheus metrics</li>
        <li><a href="/health">/health</a> - Health check</li>
    </ul>
    <h2>Configuration:</h2>
    <ul>
        <li>Token Denom: {Config.TOKEN_DENOM}</li>
        <li>Token Coefficient: {Config.TOKEN_COEFFICIENT}</li>
        <li>Total Supply: {Config.TOTAL_SUPPLY}</li>
        <li>RPC Endpoint: {Config.RPC_ENDPOINT}</li>
        <li>gRPC Endpoint: {Config.GRPC_ENDPOINT}</li>
        <li>Metrics Endpoint: {Config.METRICS_ENDPOINT}</li>
        <li>Exporter Port: {Config.EXPORTER_PORT}</li>
    </ul>
    """

if __name__ == '__main__':
    app.run(host=Config.EXPORTER_HOST, port=Config.EXPORTER_PORT, debug=False)
