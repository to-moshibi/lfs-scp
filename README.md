# Setup
1. Download lfs.scp.exe from Release
2. Place it in "C:\Windows\System32"
3. Go to your directory
```sh
git init
git lfs install
git lfs track "*.png"
git config lfs.standalonetransferagent lfs-scp
git config lfs.customtransfer.lfs-scp.path git-lfs
git config lfs.customtransfer.lfs-scp.args "example.com 22 ubuntu ~/.ssh/id_rsa"
```

# Usage

```sh
git add .
git commit -a -m "commit message"
git push
```
