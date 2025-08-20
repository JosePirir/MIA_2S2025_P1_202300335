# Comandos

## MKDISK

- mkdisk -Size=3000 -unit=K -path=/home/josepirir/Discos/Disco1.mia
hacer que que el programa no sea case sensitive
- mkdisk -path=/home/josepirir/Discos/Disco2.mia -Unit=K -size=3000
- mkdisk -size=5 -unit=M -path=/home/josepirir/Discos/Disco3.mia
- mkdisk -size=10 -path=/home/josepirir/Discos/Disco4.mia

### revisar que el -path funcione con "" y espacios

## RMDISK

- rmdisk -path=/home/josepirir/Discos/Disco4.mia

## FDISK

- fdisk -size=300 -path=/home/josepirir/Discos/Disco1.mia -name=Particion1
- fdisk -type=E -path=/home/josepirir/Discos/Disco2.mia -Unit=K -name=Particion2 -size=300
- fdisk -size=1 -type=L -unit=M -fit=BF -path=/home/josepirir/Discos/Disco3.mia -name="Particion3"
- fdisk -type=E -path=/home/josepirir/Discos/Disco2.mia -name=Part3 -Unit=K -size=200

## MOUNT

- mount -path=/home/josepirir/Discos/Disco2.mia -name=Particion2
- mount -path=/home/josepirir/Discos/Disco1.mia -name=Particion1
- mount -path=/home/josepirir/Discos/Disco1.mia -name=Particion3
- mount -path=/home/josepirir/Discos/Disco3.mia -name=Particion2

## MKFS

