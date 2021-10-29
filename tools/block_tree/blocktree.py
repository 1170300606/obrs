# -*-coding:utf-8 -*-

import requests
import sys

def block_tree(url="http://127.0.0.1:26657/block_tree"):
    odata = requests.get(url)
    jdata = odata.json()
    try:
        blocks = jdata["result"]["blocks"]
        existed = {}
        head = "graph { flow: south; }\n"
        graph = ""
        for block in blocks:
            prev = block["last_blockhash"]
            cur = block["blockhash"]
            slot = block["slot"]
            status = block["block_status"]
            val = block["validator_addr"]
            tx_num = block["tx_num"]
            
            existed[cur] = slot
            head += '''
[{}] {{ label: "
slot: {}\\n
status: {}\\n
tx_num: {}\\n
proposer_addr:\\n
{}
"}}\n
'''.format(
    slot,
    slot,
    status,
    tx_num,
    val
)
            if prev =="":
                continue
            prev_slot = existed[prev]
            graph += "[{}] -> [{}]\n".format(prev_slot, slot)
        return head+graph
    except:
        print(odata)
        print(jdata)
        return None
        
def output(data):
    if data is None:
        return
    print(data)


if __name__ == "__main__":
    data = None
    if len(sys.argv) > 1:    
        data = block_tree(url = sys.argv[1])
    else:
        data = block_tree()    
    output(data)
