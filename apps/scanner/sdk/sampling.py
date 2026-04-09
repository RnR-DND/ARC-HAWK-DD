"""
Reservoir sampling for large table scans (P4-2).

For tables with row_count > RESERVOIR_THRESHOLD, we cannot afford to read
every row into memory. Instead, Algorithm R (Knuth) produces a uniform
random sample of exactly `sample_size` rows with a single sequential pass.

Usage:
    from sdk.sampling import should_reservoir_sample, ReservoirSampler

    if should_reservoir_sample(row_count):
        sampler = ReservoirSampler(sample_size=100_000)
        for row in cursor:
            sampler.add(row)
        sampled_rows = sampler.get_sample()
    else:
        sampled_rows = list(cursor)
"""

import random
from typing import Any, Iterator, List

RESERVOIR_THRESHOLD = 1_000_000  # rows
DEFAULT_SAMPLE_SIZE = 100_000


def should_reservoir_sample(row_count: int, threshold: int = RESERVOIR_THRESHOLD) -> bool:
    """Return True if table is large enough to warrant reservoir sampling."""
    return row_count > threshold


class ReservoirSampler:
    """
    Algorithm R — single-pass reservoir sampling.
    Produces a uniform random sample of `sample_size` items from an
    arbitrarily large stream without knowing the total count in advance.

    Time: O(n), Space: O(k) where k = sample_size.
    """

    def __init__(self, sample_size: int = DEFAULT_SAMPLE_SIZE):
        self.sample_size = sample_size
        self._reservoir: List[Any] = []
        self._count = 0

    def add(self, item: Any) -> None:
        self._count += 1
        if len(self._reservoir) < self.sample_size:
            self._reservoir.append(item)
        else:
            # Replace a random element with decreasing probability
            j = random.randint(0, self._count - 1)
            if j < self.sample_size:
                self._reservoir[j] = item

    def feed(self, iterable: Iterator[Any]) -> "ReservoirSampler":
        for item in iterable:
            self.add(item)
        return self

    def get_sample(self) -> List[Any]:
        return list(self._reservoir)

    @property
    def items_seen(self) -> int:
        return self._count

    @property
    def sample_fraction(self) -> float:
        if self._count == 0:
            return 0.0
        return min(1.0, self.sample_size / self._count)
