import os
import subprocess
import CheckPython

CheckPython.ValidatePackages()

import Vulkan
import Utils
import colorama
from colorama import Fore
from colorama import Back
from colorama import Style

colorama.init()

os.chdir('../')

print(f"{Style.BRIGHT}{Back.GREEN}Setting OBLIKOVATI_DIR to {os.getcwd()}{Style.RESET_ALL}")
subprocess.call(["setx", "OBLIKOVATI_DIR", os.getcwd()])
os.environ['OBLIKOVATI_DIR'] = os.getcwd()

if (not Vulkan.CheckVulkanSDK()):
    print("Vulkan SDK not installed.")
    exit()
    
if (Vulkan.CheckVulkanSDKDebugLibs()):
    print(f"{Style.BRIGHT}{Back.GREEN}Vulkan SDK debug libs located.{Style.RESET_ALL}")

subprocess.call(["git", "lfs", "pull"])
subprocess.call(["git", "submodule", "update", "--init", "--recursive"])

if not os.path.exists("oblikovati_runtime/DotNet/"):
    os.makedirs("oblikovati_runtime/DotNet/")

print(f"{Style.BRIGHT}{Back.GREEN}Generating Visual Studio 2022 solution.{Style.RESET_ALL}")
subprocess.call(["lib_vendor_precompiled/premake/premake5.exe", "vs2022"])
