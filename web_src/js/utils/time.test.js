import {firstStartDateAfterDate, startDaysBetween, fillEmptyStartDaysWithZeroes} from './time.js';

test('startDaysBetween', () => {
  expect(startDaysBetween(new Date('2024-02-15'), new Date('2024-04-18'))).toEqual([
    1708214400000,
    1708819200000,
    1709424000000,
    1710028800000,
    1710633600000,
    1711238400000,
    1711843200000,
    1712448000000,
    1713052800000,
  ]);
});

test('firstStartDateAfterDate', () => {
  const expectedDate = new Date('2024-02-18').getTime();
  expect(firstStartDateAfterDate(new Date('2024-02-15'))).toEqual(expectedDate);

  expect(() => firstStartDateAfterDate('2024-02-15')).toThrowError('Invalid date');
});
test('fillEmptyStartDaysWithZeroes with data', () => {
  expect(fillEmptyStartDaysWithZeroes([1708214400000, 1708819200000, 1708819300000], {
    1708214400000: {'week': 1708214400000, 'additions': 1, 'deletions': 2, 'commits': 3},
    1708819200000: {'week': 1708819200000, 'additions': 4, 'deletions': 5, 'commits': 6},
  })).toEqual([
    {'week': 1708214400000, 'additions': 1, 'deletions': 2, 'commits': 3},
    {'week': 1708819200000, 'additions': 4, 'deletions': 5, 'commits': 6},
    {
      'additions': 0,
      'commits': 0,
      'deletions': 0,
      'week': 1708819300000,
    }]);
});

test('fillEmptyStartDaysWithZeroes with empty array', () => {
  expect(fillEmptyStartDaysWithZeroes([], {})).toEqual([]);
});
