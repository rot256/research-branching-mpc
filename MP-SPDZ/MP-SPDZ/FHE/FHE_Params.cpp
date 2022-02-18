
#include "FHE_Params.h"
#include "FHE/Ring_Element.h"
#include "Tools/Exceptions.h"

FHE_Params::FHE_Params(int n_mults) :
    FFTData(n_mults + 1), Chi(0.7), sec_p(-1), matrix_dim(1)
{
}

void FHE_Params::set(const Ring& R,
                     const vector<bigint>& primes)
{
  if (primes.size() != FFTData.size())
    throw runtime_error("wrong number of primes");

  for (size_t i = 0; i < FFTData.size(); i++)
    FFTData[i].init(R,primes[i]);

  set_sec(40);
}

void FHE_Params::set_sec(int sec)
{
  sec_p=sec;
  Bval=1;  Bval=Bval<<sec_p;
  Bval=FFTData[0].get_prime()/(2*(1+Bval));
  if (Bval == 0)
    throw runtime_error("distributed decryption bound is zero");
}

void FHE_Params::set_matrix_dim(int matrix_dim)
{
  assert(matrix_dim > 0);
  if (FFTData[0].get_prime() != 0)
    throw runtime_error("cannot change matrix dimension after parameter generation");
  this->matrix_dim = matrix_dim;
}

bigint FHE_Params::Q() const
{
  bigint res = FFTData[0].get_prime();
  for (size_t i = 1; i < FFTData.size(); i++)
    res *= FFTData[i].get_prime();
  return res;
}

void FHE_Params::pack(octetStream& o) const
{
  o.store(FFTData.size());
  for(auto& fd: FFTData)
    fd.pack(o);
  Chi.pack(o);
  Bval.pack(o);
  o.store(sec_p);
  o.store(matrix_dim);
}

void FHE_Params::unpack(octetStream& o)
{
  size_t size;
  o.get(size);
  FFTData.resize(size);
  for (auto& fd : FFTData)
    fd.unpack(o);
  Chi.unpack(o);
  Bval.unpack(o);
  o.get(sec_p);
  o.get(matrix_dim);
}

bool FHE_Params::operator!=(const FHE_Params& other) const
{
  if (FFTData != other.FFTData or Chi != other.Chi or sec_p != other.sec_p
      or Bval != other.Bval)
    {
      return true;
    }
  else
    return false;
}
