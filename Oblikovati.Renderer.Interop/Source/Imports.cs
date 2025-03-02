using System.Runtime.InteropServices;

namespace Oblikovati.Renderer.Interop;

internal static class Native
{
#if __MACOS__
	private const string LibraryName = "Oblikovati.Renderer.dylib";
#elif __LINUX__
	private const string LibraryName = "Oblikovati.Renderer.so";
#else
	private const string LibraryName = "Oblikovati.Renderer.dll";
#endif

	[StructLayout(LayoutKind.Sequential)]
	public struct CSharpCallbacks
	{

	}

	[StructLayout(LayoutKind.Sequential)]
	public struct CppCallbacks
	{

	}


	[DllImport(LibraryName, CallingConvention = CallingConvention.Cdecl)]
	public static extern int Initialize();

	[DllImport(LibraryName, CallingConvention = CallingConvention.Cdecl)]
	public static extern void RegisterCSharpCallbacks(ref CSharpCallbacks callbacks);

	[DllImport(LibraryName, CallingConvention = CallingConvention.Cdecl)]
	public static extern void RegisterCppCallbacks(ref CppCallbacks callbacks);
}
