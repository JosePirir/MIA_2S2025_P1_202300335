# Comandos

## MKDISK

- mkdisk -size=30 -unit=M -path=/home/josepirir/Discos/Disco1.mia
hacer que que el programa no sea case sensitive
- mkdisk -path=/home/josepirir/Discos/Disco2.mia -Unit=K -size=3000
- mkdisk -size=5 -unit=M -path=/home/josepirir/Discos/Disco3.mia
- mkdisk -size=10 -path=/home/josepirir/Discos/Disco4.mia

### revisar que el -path funcione con "" y espacios

## RMDISK

- rmdisk -path=/home/josepirir/Discos/Disco4.mia

## FDISK

- fdisk -size=3000 -path=/home/josepirir/Discos/Disco1.mia -name=Particion1
- fdisk -type=E -path=/home/josepirir/Discos/Disco2.mia -Unit=K -name=Particion2 -size=300
- fdisk -size=1 -type=L -unit=M -fit=BF -path=/home/josepirir/Discos/Disco3.mia -name="Particion3"
- fdisk -type=E -path=/home/josepirir/Discos/Disco2.mia -name=Part3 -Unit=K -size=200

## MOUNT

- mount -path=/home/josepirir/Discos/Disco2.mia -name=Particion2
- mount -path=/home/josepirir/Discos/Disco1.mia -name=Particion1
- mount -path=/home/josepirir/Discos/Disco1.mia -name=Particion3
- mount -path=/home/josepirir/Discos/Disco3.mia -name=Particion2

## MOUNTED
- mounted

## MKFS

- mkfs -id=351A
- mkfs -id=352A

## LOGIN
- login -user=root -pass=123 -id=351A

## LOGOUT
- logout

## CAT
- cat -file=/users.txt
- Tomar en cuenta que file no lleva comillas de momento



# TEST
- mkdisk -size=100 -unit=m -path=/home/josepirir/Discos/DiscoPrueba.mia
- fdisk -size=50 -unit=m -path=/home/josepirir/Discos/DiscoPrueba.mia -name=Particion1 -type=p
- mount -path=/home/josepirir/Discos/DiscoPrueba.mia -name=Particion1
- mkfs -type=full -id=351A
- login -user=root -pass=123 -id=351A
- cat -file=/users.txt