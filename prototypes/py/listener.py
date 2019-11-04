
from common.mqueue import *

def process_msg(msg):
    print(msg)
    pass

def main():
    reader = MQReader("host.packages")
    reader.listen(process_msg)
    pass
