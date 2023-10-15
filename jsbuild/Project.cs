using System;
using System.Collections.Generic;
using System.Xml;
using System.IO;

namespace JSBuild
{
    public class Project
    {
        private XmlDocument doc;
        private XmlElement root;
        private string fileName;
        private string origXml;
		private string fileFilter;
        private DirectoryInfo projectDir;
        private static readonly Project instance = new Project();

        public DirectoryInfo ProjectDir
        {
            get { return projectDir; }
            set { projectDir = value; }
        }

		public string FileFilter
		{
			//This is used to compare against Options.Files after an Options update so
			//that we know which files to remove from the tree based on the new filter.
			get { return fileFilter; }
			set { fileFilter = value; }
		}

        private Project()
        {

        }

        public static Project GetInstance()
        {
            return instance;
        }

        public void Load(string filePath)//, string fileName)
        {
            this.fileName = filePath;
            this.projectDir = new FileInfo(fileName).Directory;
            if (!String.IsNullOrEmpty(fileName) && File.Exists(fileName))
            {
                doc = new XmlDocument();
                doc.Load(fileName);
            }
            else
            {
                throw new Exception("JSB file does not exist!");
            }


            doc.Save(fileName);
            origXml = doc.InnerXml;
            root = (XmlElement)doc.GetElementsByTagName("project")[0];
            /*OnNameChanged();
            OnAuthorChanged();
            OnVersionChanged();
            OnCopyrightChanged();
            OnOutputChanged();
            OnSourceChanged();
            OnSourceDirChanged();
            OnMinifyChanged();
            OnMinDirChanged();
            OnDocChanged();
            OnDocDirChanged();*/
        }

        public List<Target> GetTargets(bool loadIncs)
        {
            List<Target> results = new List<Target>();
            XmlNodeList targets = root.GetElementsByTagName("target");
            foreach (XmlElement node in targets)
            {
                var targetName = node.GetAttribute("name");
                var targetFilePath = Util.CleanPath(node.GetAttribute("file"));
                Target t = new Target(targetName, targetFilePath, null);
                t.Debug = "True".Equals(node.GetAttribute("debug"));
                t.Shorthand = "True".Equals(node.GetAttribute("shorthand"));
                t.ShorthandList = node.GetAttribute("shorthand-list");
                if (loadIncs)
                {
                    XmlNodeList incs = node.GetElementsByTagName("include");
                    foreach (XmlNode inc in incs)
                    {
                        string iname = Util.CleanPath(inc.Attributes.GetNamedItem("name").Value);
                        //Console.WriteLine($"    *include name - {iname}:{IsSelected(iname)}:{FileExists(iname)}");
                        // IsSelected????????
                        // NOTE: use the value from xml so the values match for the IsSelected
                        if (IsSelected(inc.Attributes.GetNamedItem("name").Value) && FileExists(iname))
                        //if (FileExists(iname))
                        {
                            t.Add(iname);
                        }
                        else
                        {
                            Console.WriteLine($"Warning! Skipping including file {iname} because it did not exist in the copy over.");
                            Console.WriteLine($"    *include name - {iname}:\n        Has 'file' node? {IsSelected(iname)}\n        Exists? {FileExists(iname)}");
                        }
                    }
                }
                results.Add(t);
            }
            return results;
        }

        public bool IsSelected(string path)
        {
            //Console.WriteLine(path);
            //Console.WriteLine(root.SelectSingleNode("//project/file[@name='" + path + "']"));
            //return (root.SelectSingleNode("file[@name='" + path + "']") != null);
            /*Console.WriteLine(path);
            foreach(System.Xml.XmlElement x in root.ChildNodes) 
            {
                if (x.Attributes["name"] != null && x.Name == "file") 
                {
                    Console.WriteLine(x.Attributes["name"].Value);
                }
                
            }*/
            //throw new Exception("fart");
            //return true;
            return (root.SelectSingleNode("file[@name='" + path + "']") != null);
        }

        public bool FileExists(string file)
        {
            string path = projectDir.FullName;
            path = Util.FixPath(path);
            return new FileInfo(path + file).Exists;
        }

        internal Dictionary<string, SourceFile> LoadSourceFiles()
        {
            string path = projectDir.FullName;
            path = Util.FixPath(path);
            XmlNodeList nodes = root.GetElementsByTagName("file");
            Dictionary<string, SourceFile> files = new Dictionary<string, SourceFile>();
            foreach (XmlElement el in nodes)
            {
                // Util.FixPath adds an ending slash
                var pathAttributeValue = Util.CleanPath(el.GetAttribute("path"));
                var nameAttributeValue = Util.CleanPath(el.GetAttribute("name"));

                var filePath = path + nameAttributeValue;//el.GetAttribute("name");
                filePath = Util.CleanPath(filePath);
                FileInfo f = new FileInfo(filePath);
                if (f.Exists)
                {
					files.Add(
                        nameAttributeValue,
                        SourceFileFactory.GetSourceFile(f, pathAttributeValue)
                    );
                }
                else
                {
                    //Console.WriteLine($"File does not exist: {filePath}");
                }
            }
            return files;
        }

        public string Name
        {
            get
            {
                return root.GetAttribute("name");
            }
            set
            {
                root.SetAttribute("name", value);
            }
        }

        public string Author
        {
            get
            {
                return root.GetAttribute("author");
            }
            set
            {
                root.SetAttribute("author", value);
            }
        }
        
        public string Version
        {
            get
            {
                return root.GetAttribute("version");
            }
            set
            {
                root.SetAttribute("version", value);
            }
        }

        public string Copyright
        {
            get
            {
                return root.GetAttribute("copyright");
            }
            set
            {
                root.SetAttribute("copyright", value);
            }
        }
        
        public string Output
        {
            get
            {
                if(root.GetAttribute("output").Length < 1)
                {
                    var path = Util.FixPath(projectDir.FullName);
                    root.SetAttribute(
                        "output",
                        path + @"build\"
                        //projectDir.FullName + (projectDir.FullName.EndsWith("\\") ? "" : "\\") + @"build\"
                    );
                }
                return root.GetAttribute("output");
            }
            set
            {
                root.SetAttribute("output", value);
            }
        }
        
        public bool Source
        {
            get
            {
                return Boolean.Parse(root.GetAttribute("source"));
            }
            set
            {
                root.SetAttribute("source", value.ToString());
            }
        }
        
        public string SourceDir
        {
            get
            {
                return Util.CleanPath(root.GetAttribute("source-dir"));
            }
            set
            {
                root.SetAttribute("source-dir", value);
            }
        }

        public bool Minify
        {
            get
            {
                return Boolean.Parse(root.GetAttribute("minify"));
            }
            set
            {
                root.SetAttribute("minify", value.ToString());
            }
        }
        
        public string MinDir
        {
            get
            {
                return Util.CleanPath(root.GetAttribute("min-dir"));
            }
            set
            {
                root.SetAttribute("min-dir", value);
            }
        }
        
        public bool Doc
        {
            get
            {
                return Boolean.Parse(root.GetAttribute("doc"));
            }
            set
            {
                root.SetAttribute("doc", value.ToString());
            }
        }
        
        public string DocDir
        {
            get
            {
                return Util.CleanPath(root.GetAttribute("doc-dir"));
            }
            set
            {
                root.SetAttribute("doc-dir", value);
            }
        }
    }
}
