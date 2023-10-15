using System;
using System.Collections.Generic;
using System.Text;
using System.Diagnostics;
using CommandLine;

namespace JSBuild
{
	class Program
	{
		public static bool verbose = false;
		public static bool? cleanOutputDir = null;
		//public static string dockerFile = String.Empty;

        public class ArgumentOptions
        {
            [Option('v', "verbose", Required = false, Default = true, HelpText = "Set output to verbose messages.")]
            public bool Verbose { get; set; }

            [Option('c', "clean", Required = false, HelpText = "Delete any existing output files and folders prior to building.")]
            public bool Clean { get; set; }

			[Option('f', "jsb", Required = true, HelpText = "Path to an existing .jsb file.")]
            public string JSBPath { get; set; }

			//[Option('d', "jsdocdockerfile", Required = true, HelpText = "Path to the jsdoc dockerfile.")]
            //public string JSDocDockerFile { get; set; }

			[Option('o', "out", Required = false, HelpText = "Override the build output path specified in the project file with a new path.")]
            public string Out { get; set; }
        }

		static void RunArgumentOptions(ArgumentOptions opts)
		{
			verbose = opts.Verbose;
			cleanOutputDir = opts.Clean;
			//dockerFile = opts.JSDocDockerFile;

			try
			{
				Build(opts); //opts.JSBPath, opts.Out);
			}
			catch (Exception ex)
			{
				Console.Out.WriteLine("\nBUILD FAILED!\n" + ex.ToString());
				Wait();

				// Exit with a negative return code so that external apps like NAnt 
				// can interpret the build as unsuccessful
				Environment.Exit(-99);
			}
		}

		static void HandleParseError(IEnumerable<Error> errs)
		{
			foreach(var x in errs)
			{
				Console.WriteLine($"{x}");
			}
		}

		static void Main(string[] args)
		{
			CommandLine.Parser.Default.ParseArguments<ArgumentOptions>(args)
				.WithParsed(RunArgumentOptions)
				.WithNotParsed(HandleParseError);
		}

		static void Build(ArgumentOptions opts)
		{
			Console.Out.WriteLine("\nBuilding: " + opts.JSBPath);
			Console.Out.WriteLine();

			//string appExePath = System.Reflection.Assembly.GetExecutingAssembly().Location;
			//System.IO.FileInfo fi = new System.IO.FileInfo(appExePath);
			//appExePath = Util.FixPath(fi.DirectoryName);

			//NOTE: In order to load an existing settings.xml file for debugging, you must first
			//create one using the GUI then copy it into the directory from which this assembly
			//is running (normally jsbuildconsole\bin\debug).  Otherwise it will simply use defaults.
			
			// changed to project path
			//Options.GetInstance().Load(appExePath);
			Options.GetInstance();//.Load(projectPath);

			//If the clean flag was specified as a param, override the project setting
			Options.GetInstance().ClearOutputDir = opts.Clean;

			ProjectBuilder.MessageAvailable += new MessageDelegate(ProjectBuilder_MessageAvailable);
			ProjectBuilder.ProgressUpdate += new ProgressDelegate(ProjectBuilder_ProgressUpdate);
			ProjectBuilder.BuildComplete += new BuildCompleteDelegate(ProjectBuilder_BuildComplete);

			Project project = Project.GetInstance();
			project.Load(opts.JSBPath);

			ProjectBuilder.Build(project, opts.Out);

			Wait();
		}

		static void ProjectBuilder_BuildComplete()
		{
			Console.Out.WriteLine("\nBuild completed successfully!");
		}

		static void ProjectBuilder_ProgressUpdate(ProgressInfo progressInfo)
		{
			if (verbose) Console.Out.WriteLine(progressInfo.Message);
		}

		static void ProjectBuilder_MessageAvailable(Message message)
		{
			switch (message.Type)
			{
				case MessageTypes.Info:
					if (verbose) Console.Out.WriteLine("INFO: " + message.Text);
					break;

				case MessageTypes.Error:
					// Always write errors even if verbose = false
					Console.Out.WriteLine("\nBUILD FAILED!\n" + message.Text);
					break;

				case MessageTypes.Status:
					if (verbose) Console.Out.WriteLine(message.Text);
					break;
			}
		}

		static void Wait()
		{
            // Pause here so we can verify the output.  This hard-coded flag should be set to
            // true when debugging in Visual Studio, otherwise leave it false.  Originally I had this
            // as a #DEBUG conditional, but we commonly release Debug builds right now, and we don't want
            // this code executing for someone who's not actively debugging the console.
            //Console.Out.WriteLine("\nPress ENTER to continue...");
            //Console.ReadLine();
        }
	}
}
