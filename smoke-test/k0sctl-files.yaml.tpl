apiVersion: k0sctl.k0sproject.io/v1beta1
kind: cluster
spec:
  hosts:
    - role: controller
      uploadBinary: true
      ssh:
        address: "127.0.0.1"
        port: 9022
        keyPath: ./id_rsa_k0s
      files:
        - name: single file
          src: ./upload/toplevel.txt
          dst: /root/singlefile/renamed.txt
        - name: dest_dir
          src: ./upload/toplevel.txt
          dstDir: /root/singlefile
        - name: dir
          src: ./upload
          dstDir: /root/dir
        - name: glob
          src: ./upload/**/*.txt
          dstDir: /root/glob
        - name: url
          src: https://api.github.com/repos/k0sproject/k0s/releases
          dst: /root/url/releases.json
        - name: url
          src: https://api.github.com/repos/k0sproject/k0s/releases
          dst: /root/url
    - role: worker
      uploadBinary: true
      ssh:
        address: "127.0.0.1"
        port: 9023
        keyPath: ./id_rsa_k0s
  k0s:
    version: "$K0S_VERSION"
    config:
      spec:
        telemetry:
          enabled: false
