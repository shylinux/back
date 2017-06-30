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
	size int64
	hash string
	name string
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
func show(meta []*Meta, file string, base string) ([]*Meta, error) { // {{{

	err := os.Chdir(file)
	if err != nil {
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if base == "" {
		base = cwd
		cwd = ""
	} else {
		cwd = cwd[len(base)+1:] + "/"
	}

	list, err := ioutil.ReadDir("./")
	if err != nil {
		return nil, err
	}

	for _, v := range list {
		if v.Name()[0] == '.' {
			continue
		}

		if v.IsDir() {
			if v.Name() != "." && v.Name() != ".." {
				if meta, err = show(meta, v.Name(), base); err != nil {
					return nil, err
				}
			}
			continue
		}

		f, err := os.Open(v.Name())
		if err != nil {
			return nil, err
		}

		m := new(Meta)
		m.size = v.Size()
		allsize += m.size
		if m.size > maxsize {
			maxsize = m.size
		}

		if !*istime {
			h := md5.New()
			if _, err := io.Copy(h, f); err != nil {
				f.Close()
				return nil, err
			}
			b := h.Sum(nil)
			fmt.Print(".")
			m.hash = hex.EncodeToString(b)
		}

		m.flag = '-'
		m.time = v.ModTime()
		m.name = fmt.Sprintf("%s%s", cwd, v.Name())
		meta = append(meta, m)
		f.Close()
	}

	if err = os.Chdir("../"); err != nil {
		return nil, err
	}

	return meta, nil
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
		fmt.Printf("%s dst path %s doesn't exist\n", time.Now().Format("15:04:05"), dst)
		err = os.MkdirAll(dst, os.ModePerm)
		err_exit(err, "%s", err)
		fmt.Printf("%s create dst path %s\n", time.Now().Format("15:04:05"), dst)
	}

	if e := os.Chdir(flag.Arg(1)); e != nil {
		fmt.Printf("dst %s : %s\n", flag.Arg(1), e)
		os.Exit(1)
	} else {
		dst, _ = os.Getwd()
	}

	allsize = 0
	fmt.Printf("%s sum src(%s): ", time.Now().Format("15:04:05"), src)
	srcmeta, err := show(make([]*Meta, 0), src, "")
	err_exit(err, "%s", err)
	fmt.Printf(" %d files %s bytes\n", len(srcmeta), sizes(allsize))

	allsize = 0
	fmt.Printf("%s sum dst(%s): ", time.Now().Format("15:04:05"), dst)
	dstmeta, err := show(make([]*Meta, 0), dst, "")
	err_exit(err, "%s", err)
	fmt.Printf(" %d files %s bytes\n", len(dstmeta), sizes(allsize))

	for maxsize > 0 {
		maxbit++
		maxsize /= 10
	}

	allsize = 0
	err = back(srcmeta, dstmeta, false)
	err_exit(err, "%s", err)
	err = back(srcmeta, dstmeta, true)
	err_exit(err, "%s", err)
}

// }}}
