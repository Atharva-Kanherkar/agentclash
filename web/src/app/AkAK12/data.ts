/**
 * Religious composition of India by census year.
 *
 * Sources:
 *   - Census of India (1881-2011), Office of the Registrar General & Census
 *     Commissioner, India. https://censusindia.gov.in
 *   - Davis, Kingsley (1951). "The Population of India and Pakistan",
 *     Princeton University Press — for cleaned subcontinent figures
 *     (excluding Burma/Aden) for 1881-1941.
 *   - Pew Research Center (2021). "Religious Composition of India".
 *
 * Scope note:
 *   - 1881-1941 figures are for British India (the undivided subcontinent,
 *     including present-day Pakistan and Bangladesh, and excluding Burma
 *     which separated in 1937).
 *   - 1951-2011 figures are for the Republic of India only. The sharp drop
 *     in the Muslim share between 1941 and 1951 is the demographic effect
 *     of the 1947 Partition, not migration alone.
 *   - 1981 census excluded Assam (insurgency); 1991 census excluded Jammu
 *     & Kashmir. Reported percentages adjust for the missing states.
 *   - "Other" includes tribal/animist religions, Zoroastrians (Parsis),
 *     Jews, Baha'i, religion-not-stated, and (in 2011) "no religion".
 *
 * All values are percentage of total enumerated population.
 * Rows are sorted chronologically. Sums may differ from 100 by ±0.1
 * because of independent rounding in original sources.
 */

export type Religion =
  | "hindu"
  | "muslim"
  | "christian"
  | "sikh"
  | "buddhist"
  | "jain"
  | "other";

export type CensusRow = {
  year: number;
  scope: "british-india" | "republic-of-india";
  hindu: number;
  muslim: number;
  christian: number;
  sikh: number;
  buddhist: number;
  jain: number;
  other: number;
  /** Total enumerated population in millions (rounded). */
  populationMillions: number;
  note?: string;
};

export const CENSUS_DATA: CensusRow[] = [
  {
    year: 1881,
    scope: "british-india",
    hindu: 75.1,
    muslim: 19.97,
    christian: 0.71,
    sikh: 0.73,
    buddhist: 0.1,
    jain: 0.49,
    other: 2.9,
    populationMillions: 253,
  },
  {
    year: 1891,
    scope: "british-india",
    hindu: 72.32,
    muslim: 20.41,
    christian: 0.79,
    sikh: 0.74,
    buddhist: 0.1,
    jain: 0.49,
    other: 5.15,
    populationMillions: 287,
  },
  {
    year: 1901,
    scope: "british-india",
    hindu: 72.93,
    muslim: 21.16,
    christian: 0.99,
    sikh: 0.78,
    buddhist: 0.1,
    jain: 0.5,
    other: 3.54,
    populationMillions: 294,
  },
  {
    year: 1911,
    scope: "british-india",
    hindu: 71.49,
    muslim: 21.8,
    christian: 1.21,
    sikh: 0.83,
    buddhist: 0.1,
    jain: 0.42,
    other: 4.15,
    populationMillions: 315,
  },
  {
    year: 1921,
    scope: "british-india",
    hindu: 70.74,
    muslim: 22.24,
    christian: 1.49,
    sikh: 0.94,
    buddhist: 0.1,
    jain: 0.4,
    other: 4.09,
    populationMillions: 318,
  },
  {
    year: 1931,
    scope: "british-india",
    hindu: 70.0,
    muslim: 23.16,
    christian: 1.79,
    sikh: 1.13,
    buddhist: 0.1,
    jain: 0.37,
    other: 3.45,
    populationMillions: 353,
  },
  {
    year: 1941,
    scope: "british-india",
    hindu: 69.46,
    muslim: 24.28,
    christian: 1.91,
    sikh: 1.46,
    buddhist: 0.1,
    jain: 0.37,
    other: 2.42,
    populationMillions: 389,
    note: "Last census of undivided British India before the 1947 Partition.",
  },
  {
    year: 1951,
    scope: "republic-of-india",
    hindu: 84.1,
    muslim: 9.8,
    christian: 2.3,
    sikh: 1.79,
    buddhist: 0.05,
    jain: 0.46,
    other: 1.5,
    populationMillions: 361,
    note: "First census of independent India after the 1947 Partition.",
  },
  {
    year: 1961,
    scope: "republic-of-india",
    hindu: 83.45,
    muslim: 10.69,
    christian: 2.44,
    sikh: 1.79,
    buddhist: 0.74,
    jain: 0.46,
    other: 0.43,
    populationMillions: 439,
  },
  {
    year: 1971,
    scope: "republic-of-india",
    hindu: 82.73,
    muslim: 11.21,
    christian: 2.6,
    sikh: 1.89,
    buddhist: 0.7,
    jain: 0.47,
    other: 0.4,
    populationMillions: 548,
  },
  {
    year: 1981,
    scope: "republic-of-india",
    hindu: 82.3,
    muslim: 11.75,
    christian: 2.44,
    sikh: 1.96,
    buddhist: 0.71,
    jain: 0.47,
    other: 0.37,
    populationMillions: 665,
    note: "Assam was not enumerated due to civil unrest; figures pro-rated.",
  },
  {
    year: 1991,
    scope: "republic-of-india",
    hindu: 82.0,
    muslim: 12.12,
    christian: 2.34,
    sikh: 1.94,
    buddhist: 0.76,
    jain: 0.4,
    other: 0.44,
    populationMillions: 838,
    note: "Jammu & Kashmir was not enumerated; figures pro-rated.",
  },
  {
    year: 2001,
    scope: "republic-of-india",
    hindu: 80.46,
    muslim: 13.43,
    christian: 2.34,
    sikh: 1.87,
    buddhist: 0.77,
    jain: 0.41,
    other: 0.72,
    populationMillions: 1029,
  },
  {
    year: 2011,
    scope: "republic-of-india",
    hindu: 79.8,
    muslim: 14.23,
    christian: 2.3,
    sikh: 1.72,
    buddhist: 0.7,
    jain: 0.37,
    other: 0.88,
    populationMillions: 1211,
    note: "Most recent decennial census; the 2021 census has been postponed.",
  },
];

export const RELIGIONS: ReadonlyArray<{
  key: Religion;
  label: string;
  color: string;
  description: string;
}> = [
  {
    key: "hindu",
    label: "Hindu",
    color: "#FF9933",
    description: "Saffron — also one of the colours of the national flag.",
  },
  {
    key: "muslim",
    label: "Muslim",
    color: "#138808",
    description: "Green — also one of the colours of the national flag.",
  },
  {
    key: "christian",
    label: "Christian",
    color: "#5B8DEF",
    description: "Predominantly in Kerala, Goa, and the North-East.",
  },
  {
    key: "sikh",
    label: "Sikh",
    color: "#E0C200",
    description: "Concentrated in Punjab; born in 15th-century India.",
  },
  {
    key: "buddhist",
    label: "Buddhist",
    color: "#8E2DE2",
    description: "Indigenous to India; revived through B. R. Ambedkar.",
  },
  {
    key: "jain",
    label: "Jain",
    color: "#F5F5F5",
    description: "An ancient Indian dharmic tradition of non-violence.",
  },
  {
    key: "other",
    label: "Other / Not stated",
    color: "#9CA3AF",
    description:
      "Tribal religions, Parsis, Jews, Baha'i, none, and not-stated.",
  },
];
