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

# Validator 메트릭 (단일 밸리데이터)
cosmos_validator_delegations = Gauge('cosmos_validator_delegations', 'Delegations of the Cosmos-based blockchain validator', ['address', 'denom'], registry=registry)
cosmos_validator_tokens = Gauge('cosmos_validator_tokens', 'Validator tokens', ['address', 'denom'], registry=registry)
cosmos_validator_delegators_shares = Gauge('cosmos_validator_delegators_shares', 'Validator delegators shares', ['address', 'denom'], registry=registry)
cosmos_validator_commission_rate = Gauge('cosmos_validator_commission_rate', 'Validator commission rate', ['address'], registry=registry)
cosmos_validator_commission = Gauge('cosmos_validator_commission', 'Validator commission', ['address'], registry=registry)
cosmos_validator_rewards = Gauge('cosmos_validator_rewards', 'Validator rewards', ['address', 'denom'], registry=registry)
cosmos_validator_unbondings = Gauge('cosmos_validator_unbondings', 'Validator unbondings', ['address', 'denom'], registry=registry)
cosmos_validator_redelegations = Gauge('cosmos_validator_redelegations', 'Validator redelegations', ['address', 'denom'], registry=registry)
cosmos_validator_missed_blocks = Gauge('cosmos_validator_missed_blocks', 'Validator missed blocks', ['address'], registry=registry)
cosmos_validator_rank = Gauge('cosmos_validator_rank', 'Validator rank', ['address'], registry=registry)
cosmos_validator_active = Gauge('cosmos_validator_active', 'Validator active status', ['address'], registry=registry)
cosmos_validator_status = Gauge('cosmos_validator_status', 'Validator status', ['address'], registry=registry)
cosmos_validator_jailed = Gauge('cosmos_validator_jailed', 'Validator jailed status', ['address'], registry=registry)

# Validators 메트릭 (밸리데이터 세트)
cosmos_validators_commission = Gauge('cosmos_validators_commission', 'Commission of the Cosmos-based blockchain validator', ['address'], registry=registry)
cosmos_validators_status = Gauge('cosmos_validators_status', 'Status of the Cosmos-based blockchain validator', ['address'], registry=registry)
cosmos_validators_jailed = Gauge('cosmos_validators_jailed', 'Jailed status of the Cosmos-based blockchain validator', ['address'], registry=registry)
cosmos_validators_tokens = Gauge('cosmos_validators_tokens', 'Tokens of the Cosmos-based blockchain validator', ['address', 'denom'], registry=registry)
cosmos_validators_delegator_shares = Gauge('cosmos_validators_delegator_shares', 'Delegator shares of the Cosmos-based blockchain validator', ['address', 'denom'], registry=registry)
cosmos_validators_min_self_delegation = Gauge('cosmos_validators_min_self_delegation', 'Self declared minimum self delegation shares of the Cosmos-based blockchain validator', ['address', 'denom'], registry=registry)
cosmos_validators_missed_blocks = Gauge('cosmos_validators_missed_blocks', 'Missed blocks of the Cosmos-based blockchain validator', ['address'], registry=registry)
cosmos_validators_rank = Gauge('cosmos_validators_rank', 'Rank of the Cosmos-based blockchain validator', ['address'], registry=registry)
cosmos_validators_active = Gauge('cosmos_validators_active', '1 if the Cosmos-based blockchain validator is in active set, 0 if no', ['address'], registry=registry)



# Params 메트릭
cosmos_params_max_validators = Gauge('cosmos_params_max_validators', 'Active set length', registry=registry)
cosmos_params_unbonding_time = Gauge('cosmos_params_unbonding_time', 'Unbonding time, in seconds', registry=registry)
cosmos_params_blocks_per_year = Gauge('cosmos_params_blocks_per_year', 'Block per year', registry=registry)
cosmos_params_goal_bonded = Gauge('cosmos_params_goal_bonded', 'Goal bonded', registry=registry)
cosmos_params_inflation_min = Gauge('cosmos_params_inflation_min', 'Min inflation', registry=registry)
cosmos_params_inflation_max = Gauge('cosmos_params_inflation_max', 'Max inflation', registry=registry)
cosmos_params_inflation_rate_change = Gauge('cosmos_params_inflation_rate_change', 'Inflation rate change', registry=registry)
cosmos_params_downtail_jail_duration = Gauge('cosmos_params_downtail_jail_duration', 'Downtime jail duration, in seconds', registry=registry)
cosmos_params_min_signed_per_window = Gauge('cosmos_params_min_signed_per_window', 'Minimal amount of blocks to sign per window to avoid slashing', registry=registry)
cosmos_params_signed_blocks_window = Gauge('cosmos_params_signed_blocks_window', 'Signed blocks window', registry=registry)
cosmos_params_slash_fraction_double_sign = Gauge('cosmos_params_slash_fraction_double_sign', '% of tokens to be slashed if double signing', registry=registry)
cosmos_params_slash_fraction_downtime = Gauge('cosmos_params_slash_fraction_downtime', '% of tokens to be slashed if downtime', registry=registry)
cosmos_params_base_proposer_reward = Gauge('cosmos_params_base_proposer_reward', 'Base proposer reward', registry=registry)
cosmos_params_bonus_proposer_reward = Gauge('cosmos_params_bonus_proposer_reward', 'Bonus proposer reward', registry=registry)
cosmos_params_community_tax = Gauge('cosmos_params_community_tax', 'Community tax', registry=registry)

# General 메트릭
cosmos_general_bonded_tokens = Gauge('cosmos_general_bonded_tokens', 'Bonded tokens', registry=registry)
cosmos_general_not_bonded_tokens = Gauge('cosmos_general_not_bonded_tokens', 'Not bonded tokens', registry=registry)
cosmos_general_community_pool = Gauge('cosmos_general_community_pool', 'Community pool', registry=registry)
cosmos_general_supply_total = Gauge('cosmos_general_supply_total', 'Total supply', registry=registry)
cosmos_general_inflation = Gauge('cosmos_general_inflation', 'Inflation', registry=registry)
cosmos_general_annual_provisions = Gauge('cosmos_general_annual_provisions', 'Annual provisions', registry=registry)

# Chain 메트릭
cosmos_chain_height = Gauge('cosmos_chain_height', 'Current chain height', registry=registry)
cosmos_chain_syncing = Gauge('cosmos_chain_syncing', 'Chain syncing status', registry=registry)
cosmos_chain_latest_block_time = Gauge('cosmos_chain_latest_block_time', 'Latest block time', registry=registry)
cosmos_chain_latest_block_hash = Gauge('cosmos_chain_latest_block_hash', 'Latest block hash', registry=registry)
cosmos_chain_earliest_block_height = Gauge('cosmos_chain_earliest_block_height', 'Earliest block height', registry=registry)
cosmos_chain_earliest_block_time = Gauge('cosmos_chain_earliest_block_time', 'Earliest block time', registry=registry)

# Network 메트릭
cosmos_network_chain_id = Gauge('cosmos_network_chain_id', 'Network chain ID', registry=registry)
cosmos_network_node_id = Gauge('cosmos_network_node_id', 'Node ID', registry=registry)
cosmos_network_moniker = Gauge('cosmos_network_moniker', 'Node moniker', registry=registry)
cosmos_network_version = Gauge('cosmos_network_version', 'Node version', registry=registry)

def convert_u0g_to_0g(u0g_amount):
    return u0g_amount / Config.TOKEN_COEFFICIENT

def convert_0g_to_u0g(og_amount):
    return og_amount * Config.TOKEN_COEFFICIENT

def get_validator_info():
    try:
        validator_address = Config.VALIDATOR_ADDRESS
        
        # RPC에서 해당 validator 정보 가져오기
        response = requests.get(f"{Config.RPC_ENDPOINT}/validators", timeout=Config.METRICS_TIMEOUT)
        if response.status_code == 200:
            data = response.json()
            if 'result' in data and 'validators' in data['result']:
                validators = data['result']['validators']
                
                # config에서 지정한 validator 찾기
                target_validator = None
                for validator in validators:
                    if validator.get('address') == validator_address:
                        target_validator = validator
                        break
                
                if target_validator:
                    voting_power_u0g = int(target_validator.get('voting_power', 0))
                    
                    cosmos_validator_tokens.labels(address=validator_address, denom=Config.TOKEN_DENOM).set(voting_power_u0g)
                    cosmos_validator_delegators_shares.labels(address=validator_address, denom=Config.TOKEN_DENOM).set(voting_power_u0g)
                    cosmos_validator_commission_rate.labels(address=validator_address).set(0.0)
                    cosmos_validator_commission.labels(address=validator_address).set(0.0)
                    cosmos_validator_missed_blocks.labels(address=validator_address).set(0.0)
                    cosmos_validator_rank.labels(address=validator_address).set(1.0)
                    cosmos_validator_active.labels(address=validator_address).set(1.0)
                    cosmos_validator_status.labels(address=validator_address).set(1.0)
                    cosmos_validator_jailed.labels(address=validator_address).set(0.0)
                    
                    cosmos_validator_delegations.labels(address=validator_address, denom=Config.TOKEN_DENOM).set(voting_power_u0g)
                    cosmos_validator_rewards.labels(address=validator_address, denom=Config.TOKEN_DENOM).set(0.0)
                    cosmos_validator_unbondings.labels(address=validator_address, denom=Config.TOKEN_DENOM).set(0.0)
                    cosmos_validator_redelegations.labels(address=validator_address, denom=Config.TOKEN_DENOM).set(0.0)
                    
                    return validator_address
                else:
                    print(f"Validator {validator_address} not found in validators list")
            else:
                print("No validators found in response")
        else:
            print(f"Failed to get validators: {response.status_code}")
        
    except Exception as e:
        print(f"Error getting validator info: {e}")
    return None

def get_validators_set():
    try:
        # RPC에서 동적으로 데이터 가져오기
        response = requests.get(f"{Config.RPC_ENDPOINT}/validators", timeout=Config.METRICS_TIMEOUT)
        if response.status_code == 200:
            data = response.json()
            if 'result' in data and 'validators' in data['result'] and len(data['result']['validators']) > 0:
                validator = data['result']['validators'][0]
                validator_address = validator.get('address', 'unknown')
                voting_power_u0g = int(validator.get('voting_power', 0))
                
                # 메트릭 설정
                cosmos_validators_commission.labels(address=validator_address).set(0.0)
                cosmos_validators_status.labels(address=validator_address).set(1.0)
                cosmos_validators_jailed.labels(address=validator_address).set(0.0)
                cosmos_validators_tokens.labels(address=validator_address, denom=Config.TOKEN_DENOM).set(voting_power_u0g)
                cosmos_validators_delegator_shares.labels(address=validator_address, denom=Config.TOKEN_DENOM).set(voting_power_u0g)
                cosmos_validators_min_self_delegation.labels(address=validator_address, denom=Config.TOKEN_DENOM).set(0.0)
                cosmos_validators_missed_blocks.labels(address=validator_address).set(0.0)
                cosmos_validators_rank.labels(address=validator_address).set(1.0)
                cosmos_validators_active.labels(address=validator_address).set(1.0)
            else:
                print("No validators found in response")
        else:
            print(f"Failed to get validators: {response.status_code}")
        
    except Exception as e:
        print(f"Error getting validators set: {e}")



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
                    cosmos_chain_height.set(float(height))
                    
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
                    # 실제 값 사용 (hash 대신)
                    cosmos_network_chain_id.set(float(hash(node_info.get('network', ''))))
                    cosmos_network_node_id.set(float(hash(node_info.get('id', ''))))
                    cosmos_network_moniker.set(float(hash(node_info.get('moniker', ''))))
                    cosmos_network_version.set(float(hash(node_info.get('version', ''))))
                
                return True
    except Exception as e:
        print(f"Error getting chain status: {e}")
    return False

def get_params_info():
    try:
        cosmos_params_max_validators.set(100)
        cosmos_params_unbonding_time.set(1814400)
        cosmos_params_blocks_per_year.set(6311520)
        cosmos_params_goal_bonded.set(0.67)
        cosmos_params_inflation_min.set(0.07)
        cosmos_params_inflation_max.set(0.20)
        cosmos_params_inflation_rate_change.set(0.13)
        cosmos_params_downtail_jail_duration.set(600)
        cosmos_params_min_signed_per_window.set(0.5)
        cosmos_params_signed_blocks_window.set(100)
        cosmos_params_slash_fraction_double_sign.set(0.05)
        cosmos_params_slash_fraction_downtime.set(0.01)
        cosmos_params_base_proposer_reward.set(0.01)
        cosmos_params_bonus_proposer_reward.set(0.04)
        cosmos_params_community_tax.set(0.02)
    except Exception as e:
        print(f"Error getting params info: {e}")

def get_general_info():
    try:
        # RPC에서 validators 정보를 가져와서 bonded tokens 계산
        response = requests.get(f"{Config.RPC_ENDPOINT}/validators", timeout=Config.METRICS_TIMEOUT)
        if response.status_code == 200:
            data = response.json()
            if 'result' in data and 'validators' in data['result']:
                validators = data['result']['validators']
                
                # 모든 validator의 voting power 합계 계산
                total_bonded_tokens = 0
                for validator in validators:
                    voting_power = int(validator.get('voting_power', 0))
                    total_bonded_tokens += voting_power
                
                total_supply_u0g = convert_0g_to_u0g(Config.TOTAL_SUPPLY)
                
                cosmos_general_bonded_tokens.set(float(total_bonded_tokens))
                cosmos_general_not_bonded_tokens.set(float(total_supply_u0g - total_bonded_tokens))
                cosmos_general_community_pool.set(0.0)
                cosmos_general_supply_total.set(float(total_supply_u0g))
                cosmos_general_inflation.set(0.07)
                cosmos_general_annual_provisions.set(70000000)
            else:
                # RPC에서 데이터를 가져올 수 없는 경우 기본값 사용
                total_supply_u0g = convert_0g_to_u0g(Config.TOTAL_SUPPLY)
                bonded_tokens_u0g = convert_0g_to_u0g(36)
                
                cosmos_general_bonded_tokens.set(bonded_tokens_u0g)
                cosmos_general_not_bonded_tokens.set(total_supply_u0g - bonded_tokens_u0g)
                cosmos_general_community_pool.set(0.0)
                cosmos_general_supply_total.set(total_supply_u0g)
                cosmos_general_inflation.set(0.07)
                cosmos_general_annual_provisions.set(70000000)
    except Exception as e:
        print(f"Error getting general info: {e}")

def get_metrics():
    try:
        response = requests.get(Config.METRICS_ENDPOINT, timeout=Config.METRICS_TIMEOUT)
        if response.status_code == 200:
            metrics_text = response.text
            
            if 'cometbft_consensus_height' in metrics_text:
                height_match = re.search(r'cometbft_consensus_height\{[^}]*\} ([0-9.]+)', metrics_text)
                if height_match:
                    height = float(height_match.group(1))
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
    
    get_params_info()
    get_general_info()
    get_metrics()
    
    return Response(generate_latest(registry), mimetype=CONTENT_TYPE_LATEST)

@app.route('/metrics/params')
def params_metrics():
    get_params_info()
    return Response(generate_latest(registry), mimetype=CONTENT_TYPE_LATEST)

@app.route('/metrics/validators')
def validators_metrics():
    try:
        # RPC에서 동적으로 데이터 가져오기
        response = requests.get(f"{Config.RPC_ENDPOINT}/validators", timeout=Config.METRICS_TIMEOUT)
        
        if response.status_code == 200:
            data = response.json()
            
            if 'result' in data and 'validators' in data['result'] and len(data['result']['validators']) > 0:
                validator = data['result']['validators'][0]
                validator_address = validator.get('address', 'unknown')
                voting_power_u0g = int(validator.get('voting_power', 0))
                
                # 메트릭 설정
                cosmos_validators_commission.labels(address=validator_address).set(0.0)
                cosmos_validators_status.labels(address=validator_address).set(1.0)
                cosmos_validators_jailed.labels(address=validator_address).set(0.0)
                cosmos_validators_tokens.labels(address=validator_address, denom=Config.TOKEN_DENOM).set(voting_power_u0g)
                cosmos_validators_delegator_shares.labels(address=validator_address, denom=Config.TOKEN_DENOM).set(voting_power_u0g)
                cosmos_validators_min_self_delegation.labels(address=validator_address, denom=Config.TOKEN_DENOM).set(0.0)
                cosmos_validators_missed_blocks.labels(address=validator_address).set(0.0)
                cosmos_validators_rank.labels(address=validator_address).set(1.0)
                cosmos_validators_active.labels(address=validator_address).set(1.0)
            else:
                return
        else:
            return
    except Exception as e:
        print(f"Error in validators_metrics: {e}")
    
    return Response(generate_latest(registry), mimetype=CONTENT_TYPE_LATEST)

@app.route('/metrics/validator')
def validator_metrics():
    get_validator_info()
    return Response(generate_latest(registry), mimetype=CONTENT_TYPE_LATEST)



@app.route('/metrics/general')
def general_metrics():
    get_chain_status()
    get_general_info()
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
        <li><a href="/metrics">/metrics</a> - All Prometheus metrics</li>
        <li><a href="/metrics/params">/metrics/params</a> - Chain parameters</li>
        <li><a href="/metrics/validators">/metrics/validators</a> - Validators set</li>
        <li><a href="/metrics/validator">/metrics/validator</a> - Specific validator</li>

        <li><a href="/metrics/general">/metrics/general</a> - General chain metrics</li>
        <li><a href="/health">/health</a> - Health check</li>
    </ul>
    <h2>Usage Examples:</h2>
    <ul>

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
