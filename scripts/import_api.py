#!/usr/bin/env python3

import copy
import os
import json
import requests

# 获取脚本所载目录
PATH = lambda p: os.path.abspath(os.path.join(os.path.dirname(__file__), p))

def parse_json(path):
    if not os.path.exists(path):
        raise FileNotFoundError
    with open(path, "r", encoding="utf-8") as fp:
        paths_data = json.load(fp)
        paths_data_copy = copy.deepcopy(paths_data)

        for k, v in paths_data["paths"].items():
            # 先处理value
            for v_k, v_v in v.items():
                # if v_k in ["get","post","delete","put","patch"]:
                # 获取方法名
                method_name = v_v["operationId"].split("_")[1]
                paths_data_copy["paths"][k][v_k]["summary"] = method_name
                if "parameters" in paths_data_copy["paths"][k][v_k]:
                    for param in paths_data_copy["paths"][k][v_k]["parameters"]:
                        if param["in"] == "path" and "description" not in param:
                            if "pattern" in param:
                                tmp_arr = param["pattern"].split("[^/]+")
                                desc = ""
                                for index, tmp in enumerate(tmp_arr):
                                    desc += tmp
                                    if index == 0:
                                        desc += "{" + f"{tmp[0:-2]}_id" + "}"
                                    elif index > 0 and index != (len(tmp_arr) - 1):
                                        desc += "{" + f"{tmp[1:-2]}_id" + "}"
                                    else:
                                        desc += tmp
                                param["description"] = desc
    return paths_data_copy


# if payload_data["tags"][0]["name"] == "DeviceDebugService":
def yapi_request( payload_data, importtype):

    payload = {
        "json": json.dumps(payload_data),
        "type": "swagger",
        "merge": importtype,
        "token": token_branches
    }
    headers = {
        'Content-Type': 'application/x-www-form-urlencoded'
    }
    response = requests.request("POST", f"{urls[env]}{api_path}", headers=headers, data=payload).json()
    print("swagger json import result ", response)



def import_apiData(path):
    payload_data = parse_json(path)
    working_file = path.split("/")[-1:][0]
    print(f"working on swagger file {payload_data} ")
    


# def import_apis():
#     for root, dirs, files in os.walk(PATH(f"../{project_dir}"), topdown=True):
#         if root.endswith("services"):
#             for file in files:
#                 file_run_flag = file.endswith("swagger.json")
#                 # if len(include_services) > 0:
#                 #     service_name = file.split(".")[0]
#                 #     file_run_flag = (file_run_flag and service_name in include_services)
#                 if file_run_flag:
#                     swagger_filePath = os.path.join(root, file)
#                     import_apiData(swagger_filePath, import_type)



if __name__ == '__main__':
    import_type = "merge"
    # yapi openapi import url
    api_path = "/api/open/import_data"
    # buf generated swagger files location
    project_dir = "grpc/proto/openapiv2/services"
    env = "staging"
    urls = {
        "local": "http://127.0.0.1:3000",
        "staging": "https://devportal.xxx.cn/yapi"
    }
    token_branches = ''
    try:
        import_apiData("./import_api.json")

    except Exception as e:
        print(f"import gaussian apis , result in error {e}")


