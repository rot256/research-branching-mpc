#ifndef _Memory
#define _Memory

/* Class to hold global memory of our system */

#include <iostream>
#include <set>
using namespace std;

// Forward declaration as apparently this is needed for friends in templates
template<class T> class Memory;
template<class T> ostream& operator<<(ostream& s,const Memory<T>& M);
template<class T> istream& operator>>(istream& s,Memory<T>& M);

#include "Processor/Program.h"
#include "Tools/CheckVector.h"

template<class T> 
class Memory
{
  public:

  CheckVector<T> MS;
  CheckVector<typename T::clear> MC;

  void resize_s(int sz)
    { MS.resize(sz); }
  void resize_c(int sz)
    { MC.resize(sz); }

  unsigned size_s()
    { return MS.size(); }
  unsigned size_c()
    { return MC.size(); }

  template<class U>
  static void check_index(const vector<U>& M, size_t i)
    {
      if (i >= M.size())
        throw overflow("memory", i, M.size());
    }

  const typename T::clear& read_C(int i) const
    {
      check_index(MC, i);
      return MC[i];
    }
  const T& read_S(int i) const
    {
      check_index(MS, i);
      return MS[i];
    }

  void write_C(unsigned int i,const typename T::clear& x)
    {
      check_index(MC, i);
      MC[i]=x;
    }
  void write_S(unsigned int i,const T& x)
    {
      check_index(MS, i);
      MS[i]=x;
    }

  void minimum_size(RegType secret_type, RegType clear_type,
      const Program& program, const string& threadname);

  friend ostream& operator<< <>(ostream& s,const Memory<T>& M);
  friend istream& operator>> <>(istream& s,Memory<T>& M);
};

#endif

