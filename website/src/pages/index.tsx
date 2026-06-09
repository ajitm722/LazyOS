import {useEffect} from 'react';

export default function Home() {
  useEffect(() => {
    window.location.href = '/LazyOS/docs/intro';
  }, []);
  return null;
}
