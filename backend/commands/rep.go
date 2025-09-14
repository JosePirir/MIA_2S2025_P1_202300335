package commands

import "strings"


func ExecuteRep(name string, path string, id string, path_file_ls string) {	
	name = strings.ToLower(name)
	
	switch name {
		case "mbr":
			MBR(id, path)

		case "disk":
			DISK(id, path)
			
		case "inode":
			INODE(id, path)

		case "block":
			BLOCK(id, path)

		case "bm_inode":
			BM_INODE(id, path)

		case "bm_block":
			BM_BLOCK(id, path)

		case "tree":
			TREE(id, path)

		case "sb":
			SB(id, path)

		case "file":
			FILE(id, path_file_ls, path)
			
		case "ls":
			LS(id, path, path_file_ls)
		default:			
			return
	}
}