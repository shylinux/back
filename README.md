# back
backup &amp; restore important file 用于文件的备份与恢复

there are five kinds file when backup or restore files:

- **\+ new** file only in *srcpath*
- **\> newer** file in *srcpath* than in *dstpath*
- **= same** file between *srcpath* & *dstpath*
- **< older** file in *srcpath* than in *dstpath*
- **\- old** file only in *dstpath*


## usage: back [option] *srcpath* *dstpath*
1. **-save**
copy **new(+)** and **newer(>)** files from *srcpath* to *dstpath*
1. **-backup**
copy **all(+ > <)** files from *srcpath* to *dstpath*
1. **-same**
copy **all(+ > <)** files from *srcpath* to *dstpath*, move **old(-)** files from *dstpath* to *dstpath*/trash

- **-time** compare files only by time stamp, don't use hash
- **-force** copy or move files without confirm
- **-help** show help
