package main // {{{
// }}}
import ( // {{{
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"
)

// }}}

type Meta struct { // {{{
	flag rune

	hash string
	name string

	size int64
	time time.Time
}

// }}}

var ( // {{{
	ishelp          = flag.Bool("help", false, "usage: back [options] src dst")
	istime          = flag.Bool("time", false, "compare files by time stamp")
	islist          = flag.Bool("list", true, "list different(+ > < -) files")
	issave          = flag.Bool("save", false, "save new(+) and newer(>) files from src to dst")
	isload          = flag.Bool("load", false, "load new(-) and newer(<) files from dst to src")
	issame          = flag.Bool("same", false, "save all(+ > <) from src to dst delete rest(-) from dst")
	isbackup        = flag.Bool("backup", false, "backup all(+ > <) files from src to dst")
	isrecover       = flag.Bool("recover", false, "recover all(- < >) files from dst to src")
	isforce         = flag.Bool("force", false, "force copy with no confirm ")
	maxbit    int64 = 1
	maxsize   int64
	sumsize   int64
	allsize   int64
	src       string
	dst       string
)

// }}}

func confirm(format string, msg ...interface{}) bool { // {{{
	fmt.Printf(format, msg...)

	var a string
	fmt.Scanf("%s\n", &a)
	if len(a) > 0 && a[0] == 'n' {
		return false
	}
	return true
}

// }}}
func sizes(s int64) string { // {{{
	if s > 10000000000 {
		return fmt.Sprintf("%dG", s/1000000000)
	}
	if s > 10000000 {
		return fmt.Sprintf("%dM", s/1000000)
	}
	if s > 10000 {
		return fmt.Sprintf("%dK", s/1000)
	}
	return fmt.Sprintf("%dB", s)
}

// }}}

func save(size int64, src string, dst string) (err error) { // {{{
	dir := path.Dir(dst)
	if _, err = os.Stat(dir); err != nil {
		if err = os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
		fmt.Printf("%s create dst path %s\n", time.Now().Format("15:04:05"), dir)
	}
repeat:
	if !*isforce {
		fmt.Printf("%s copy %s from %s to %s y(yes/no/quit/delete/compare):", time.Now().Format("15:04:05"), sizes(size), src, dst)

		var a string
		fmt.Scanf("%s\n", &a)
		if len(a) > 0 && a[0] == 'n' {
			return nil
		}
		if len(a) > 0 && a[0] == 'q' {
			os.Exit(0)
		}
		if len(a) > 0 && a[0] == 'd' {
			return os.Remove(src)
		}

		if len(a) > 0 && a[0] == 'c' {
			cmd := exec.Command("vim", "-d", src, dst)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			if cmd.Run() != nil {
				fmt.Println("can find editor vim")
			}
			goto repeat
		}
	}

	dstf, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstf.Close()

	srcf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcf.Close()

	fmt.Printf("%s copy %s to %s ... ", time.Now().Format("15:04:05"), sizes(size), dst)
	size, err = io.Copy(dstf, srcf)
	sumsize += size
	fmt.Printf("done %%%d\n", sumsize*100/allsize)
	return nil
}

// }}}
func back(srcmeta []*Meta, dstmeta []*Meta, action bool) error { // {{{
	if *islist && !action {
		fmt.Fprintf(os.Stdout, "\n  %*s %*s\n",
			maxbit+12, "source info",
			maxbit+12, "destination info")
	}

	for _, f := range srcmeta {
		f.flag = '+'
		var ff *Meta
		for _, ff = range dstmeta {
			if f.name == ff.name {
				if (*istime && f.time.Equal(ff.time)) ||
					(!*istime && f.hash == ff.hash) {
					f.flag, ff.flag = '=', '='
				} else if f.time.Before(ff.time) {
					f.flag, ff.flag = '<', '<'
				} else {
					f.flag, ff.flag = '>', '>'
				}
				break
			}
		}

		if !action {
			allsize += f.size
			if *islist {
				if f.flag == '+' {
					fmt.Fprintf(os.Stdout, "%c %*s %s %*s %11s  %s\n", f.flag,
						maxbit, sizes(f.size), f.time.Format("01/02 15:04"),
						maxbit, " ", " ", f.name)
				} else if f.flag != '=' {
					fmt.Fprintf(os.Stdout, "%c %*s %s %*s %11s  %s\n", f.flag,
						maxbit, sizes(f.size), f.time.Format("01/02 15:04"),
						maxbit, sizes(ff.size), ff.time.Format("01/02 15:04"),
						f.name)
				}
			}
			continue
		}

		size := f.size
		srcf := path.Join(src, f.name)
		dstf := path.Join(dst, f.name)
		needcopy := false

		switch f.flag {
		case '>':
			if *issave || *isbackup || *issame {
				needcopy = true
			}
			if *isrecover {
				srcf = path.Join(dst, f.name)
				dstf = path.Join(src, f.name)
				size = ff.size
				needcopy = true
			}
		case '<':
			if *isbackup || *issame {
				needcopy = true
			}
			if *isload || *isrecover {
				srcf = path.Join(dst, f.name)
				dstf = path.Join(src, f.name)
				size = ff.size
				needcopy = true
			}
		case '+':
			if *issave || *isbackup || *issame {
				needcopy = true
			}
			if *issame && !*istime {
				for _, ff := range dstmeta {
					if ff.flag != '-' {
						continue
					}
					if f.hash == ff.hash {
						if confirm("rename %s to %s y(yes/no):", ff.name, f.name) {
							if err := os.Rename(path.Join(dst, ff.name), path.Join(dst, f.name)); err != nil {
								return err
							}
							ff.name = f.name
							ff.flag = '='
							needcopy = false
						}
					}
				}
			}
		}

		if needcopy {
			if err := save(size, srcf, dstf); err != nil {
				return err
			}
		}
	}

	for _, f := range dstmeta {
		if f.flag != '-' {
			continue
		}

		if action {
			if *issame && confirm("remove %s from %s y(yes/no):", f.name, dst) {
				if err := os.Remove(path.Join(dst, f.name)); err != nil {
					return err
				}
			}
			if *isload || *isrecover {
				if err := save(f.size, path.Join(dst, f.name), path.Join(src, f.name)); err != nil {
					return err
				}
			}
		} else {
			allsize += f.size
			if *islist {
				fmt.Fprintf(os.Stdout, "%c %*s %11s %*s %s  %s\n", f.flag,
					maxbit, " ", " ",
					maxbit, sizes(f.size), f.time.Format("01/02 15:04"),
					f.name)
			}
		}
	}

	return nil
}

// }}}

func sum(file string) string { // {{{
	if f, e := os.Open(file); e == nil {
		defer f.Close()

		h := md5.New()
		if _, e := io.Copy(h, f); e == nil {
			return hex.EncodeToString(h.Sum(nil))
		}
	}

	return ""
}

// }}}
func diff(srcmeta map[string]*Meta, dstmeta map[string]*Meta) { // {{{
	for k, v := range srcmeta {
		if vv, ok := dstmeta[k]; ok {
			if v.time.After(vv.time) {
				v.flag = '>'
				vv.flag = '>'
			} else if v.time.Before(vv.time) {
				v.flag = '<'
				vv.flag = '<'
			} else {
				v.flag = '='
				vv.flag = '='
			}

			if v.size == vv.size {
				v.hash = sum(path.Join(src, v.name))
				vv.hash = sum(path.Join(dst, vv.name))
				if v.hash == vv.hash {
					v.flag = '='
					vv.flag = '='
				} else {
					if v.flag == '=' {
						v.flag = '>'
						vv.flag = '>'
					}
				}
			}
		} else {
			v.flag = '+'
		}
	}
}

// }}}
func scan(meta map[string]*Meta, file string, base string) (m map[string]*Meta, e error) { // {{{

	list, e := ioutil.ReadDir(file)
	if e != nil {
		return
	}

	for _, v := range list {
		if v.Name()[0] == '.' {
			continue
		}

		cwd := path.Join(file, v.Name())

		if v.IsDir() {
			if v.Name() != "." && v.Name() != ".." {
				if meta, e = scan(meta, cwd, base); e != nil {
					return nil, e
				}
			}
			continue
		}

		cwd = cwd[len(base):]

		m := new(Meta)

		m.flag = '-'

		m.size = v.Size()
		allsize += m.size
		if m.size > maxsize {
			maxsize = m.size
		}

		m.time = v.ModTime()
		m.name = fmt.Sprintf("%s", cwd)
		meta[cwd] = m
		fmt.Print(".")

	}

	return meta, nil
}

// }}}
func action(done bool, srcmeta map[string]*Meta, dstmeta map[string]*Meta) { // {{{
	for _, v := range srcmeta {
		if v.flag == '+' {
			fmt.Printf("%c %d %10d %s\n", v.flag, v.time.Unix(), v.size, v.name)
		}
	}

	for _, v := range srcmeta {
		if v.flag == '>' {
			fmt.Printf("%c %d %10d %s\n", v.flag, v.time.Unix(), v.size, v.name)
		}
	}
	for _, v := range srcmeta {
		if v.flag == '=' {
			fmt.Printf("%c %d %10d %s\n", v.flag, v.time.Unix(), v.size, v.name)
		}
	}
	for _, v := range srcmeta {
		if v.flag == '<' {
			fmt.Printf("%c %d %10d %s\n", v.flag, v.time.Unix(), v.size, v.name)
		}
	}
	for _, v := range dstmeta {
		if v.flag == '-' {
			fmt.Printf("%c %d %10d %s\n", v.flag, v.time.Unix(), v.size, v.name)
		}
	}
}

// }}}

func err_exit(err error, format string, str ...interface{}) { // {{{
	if err != nil {
		fmt.Printf(format, str)
		os.Exit(1)
	}
}

// }}}
func main() { // {{{
	if flag.Parse(); flag.NArg() != 2 {
		fmt.Printf("usage %s [options] src dst\n", os.Args[0])
		os.Exit(1)
	}

	old, _ := os.Getwd()
	if e := os.Chdir(flag.Arg(0)); e != nil {
		fmt.Printf("src %s : %s\n", flag.Arg(0), e)
		os.Exit(1)
	} else {
		src, _ = os.Getwd()
	}

	os.Chdir(old)
	dst = flag.Arg(1)
	if _, err := os.Stat(dst); err != nil {
		err = os.MkdirAll(dst, os.ModePerm)
		err_exit(err, "%s", err)
		fmt.Printf("%s create dst path: %s\n", time.Now().Format("15:04:05"), dst)
	}

	if e := os.Chdir(flag.Arg(1)); e != nil {
		fmt.Printf("dst %s : %s\n", flag.Arg(1), e)
		os.Exit(1)
	} else {
		dst, _ = os.Getwd()
	}

	allsize = 0
	fmt.Printf("%s sum src(%s): ", time.Now().Format("15:04:05"), src)
	srcmeta, err := scan(make(map[string]*Meta, 0), src, src)
	err_exit(err, "%s", err)
	fmt.Printf(" %d files %s bytes\n", len(srcmeta), sizes(allsize))

	allsize = 0
	fmt.Printf("%s sum dst(%s): ", time.Now().Format("15:04:05"), dst)
	dstmeta, err := scan(make(map[string]*Meta, 0), dst, dst)
	err_exit(err, "%s", err)
	fmt.Printf(" %d files %s bytes\n", len(dstmeta), sizes(allsize))

	diff(srcmeta, dstmeta)

	for maxsize > 0 {
		maxbit++
		maxsize /= 10
	}
	allsize = 0
	action(false, srcmeta, dstmeta)
	action(true, srcmeta, dstmeta)
}

// }}}
