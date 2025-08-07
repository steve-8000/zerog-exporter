import os
from dotenv import load_dotenv

load_dotenv('config.env')

class Config:
    RPC_ENDPOINT = os.getenv('RPC_ENDPOINT', 'http://localhost:50657')
    GRPC_ENDPOINT = os.getenv('GRPC_ENDPOINT', 'localhost:50060')
    METRICS_ENDPOINT = os.getenv('METRICS_ENDPOINT', 'http://localhost:50660')
    
    EXPORTER_HOST = os.getenv('EXPORTER_HOST', '0.0.0.0')
    EXPORTER_PORT = int(os.getenv('EXPORTER_PORT', '26660'))
    
    TOKEN_DENOM = os.getenv('TOKEN_DENOM', 'u0g')
    TOKEN_COEFFICIENT = int(os.getenv('TOKEN_COEFFICIENT', '1000000000'))
    TOTAL_SUPPLY = int(os.getenv('TOTAL_SUPPLY', '1000000000'))
    

    
    METRICS_SCRAPE_INTERVAL = int(os.getenv('METRICS_SCRAPE_INTERVAL', '15'))
    METRICS_TIMEOUT = int(os.getenv('METRICS_TIMEOUT', '10'))
    METRICS_RETRY_COUNT = int(os.getenv('METRICS_RETRY_COUNT', '3'))
    
    VALIDATOR_ADDRESS = os.getenv('VALIDATOR_ADDRESS', '30535EF0D596876C5DBFCF825D64134550AB4945')
    
    LOG_LEVEL = os.getenv('LOG_LEVEL', 'info')
    LOG_FORMAT = os.getenv('LOG_FORMAT', 'json')
