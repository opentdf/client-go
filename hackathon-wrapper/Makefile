libdir = /src/virtru-tdf3-lib-cpp/lib
includedir = /src/virtru-tdf3-lib-cpp/include

libwrapper.so: wrapper.o
	g++ -shared -o libwrapper.so wrapper.o -static-libstdc++ -L$(libdir) -lvirtru_tdf3_static_combined

wrapper.o:
	g++ -shared -fPIC -Wall -Wextra -std=c++17 -c wrapper.cpp -I$(includedir)

clean:
	rm -f wrapper.o libwrapper.so
