package commands


func ExecuteRep(name string, path string, id string, path_file_ls string) {	
	switch name {
		case "mbr":
			MBR(id, path)

		case "disk":
			DISK(id, path)
			
		case "inode":
			INODE(id, path)
	}


}