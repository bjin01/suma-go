#!/usr/bin/env python3
import requests
import pprint
import time

MANAGER_URL = "https://suma1.bo2go.home/rhn/manager/api"
MANAGER_LOGIN = "admin"
MANAGER_PASSWORD = "suselinux"
SSLVERIFY = "/home/bjin/tmp/RHN-ORG-TRUSTED-SSL-CERT"

data = {"login": MANAGER_LOGIN, "password": MANAGER_PASSWORD}
response = requests.post(MANAGER_URL + '/auth/login', json=data, verify=SSLVERIFY)
print("LOGIN: {}:{}".format(response.status_code, response.json()))

cookies = response.cookies

mycookie = {}
for x in cookies:
    
    if x.name in "pxt-session-cookie":
        a = {x.name: x.value}
        print("cookies are: {}".format(a))
        headers = {'content-type': 'application/json'}
        payload = {'sessionKey': x.value}
        time.sleep(2)
        res2 = requests.get(MANAGER_URL + '/channel/listAllChannels', cookies=a, verify=SSLVERIFY)
        print("RETCODE: {}".format(res2.status_code))
        # print("json raw: {}".format(res2.json()))
        if res2.status_code != 200:
            print("Request failed. Logout...")
            res2 = requests.post(MANAGER_URL + '/auth/logout', cookies=a, verify=SSLVERIFY)
            print("RETCODE logout: {}".format(res2.status_code))
            exit
        else:
            print("res2.json {}".format(res2))
            pprint.pprint(res2.json())
            break



res2 = requests.post(MANAGER_URL + '/auth/logout', cookies=cookies, verify=SSLVERIFY)
print("RETCODE logout: {}".format(res2.status_code))
pprint.pprint(res2.json())
