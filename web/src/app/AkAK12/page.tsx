import type { Metadata } from "next";
import { Viewer } from "./viewer";

export const metadata: Metadata = {
  title: "India · Religious Demographics 1881–2011",
  description:
    "An interactive 3D globe visualisation of India's religious composition across the decennial censuses from 1881 to 2011, built from Census of India data.",
  robots: { index: false, follow: false },
};

export default function Page() {
  return <Viewer />;
}
