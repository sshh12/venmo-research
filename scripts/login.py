# pip install venmo-api==0.2.2
from venmo_api import Client


access_token = Client.get_access_token(username='...',
                                       password='...')

print(access_token)