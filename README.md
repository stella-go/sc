```bash
sc: a proxy forwarding tool

usage:
        netwnork topology:
            192.1.9.2 ----> 172.1.7.2
            192.1.9.2 <-x-- 172.1.7.2
            192.2.9.1 ----> 172.1.7.2
            192.2.9.1 <-x-- 172.1.7.2
            192.2.9.1 --x-> 192.1.9.2
            192.2.9.1 <-x-- 192.1.9.2

        except access:
            192.2.9.1 ----> 192.1.9.2:8080
        
        commands:
            [172.1.7.2]: sc -a foobar 8888 9999
            [192.1.9.2]: sc -a foobar 172.1.7.2:9999 192.1.9.2:8080
            [192.2.9.1]: curl 172.1.7.2:8888
        
          -a string
        password for authentication, required
  -c    client side
  -u    udp
  -v    version
  ```