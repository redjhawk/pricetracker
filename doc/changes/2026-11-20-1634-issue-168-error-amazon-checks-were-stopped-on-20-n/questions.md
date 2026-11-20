# Questions — issue #168

Draft specification: [Amazon human-like checks](../../specifications/amazon-human-checks/functional.md). Today, all due items are checked at 08:00 and 20:00 Paris time, one after another, with a random 5–60 s pause between two items.

1. **Time range of the two daily Amazon checks.** Within which hours (Paris time) may each scheduled check happen?
   - a) Morning check between 06:00 and 11:00, evening check between 17:00 and 23:00.
   - b) Another pair of ranges you give (e.g. 07:00–12:00 and 18:00–22:00).
   - c) Any time of the day, with at least N hours between the two checks (give N).

2. **LeBoncoin.** Do the real web browser and the random check times also apply to LeBoncoin items?
   - a) No, Amazon only; LeBoncoin keeps its current behaviour.
   - b) Random check times for LeBoncoin too, but no browser.
   - c) Both the browser and random check times for LeBoncoin too.

3. **Spreading of items.** How are the tracked Amazon items checked within a scheduled check?
   - a) All items together in one run that starts at the random time, one after another with the current random 5–60 s pause.
   - b) As a), but with longer random pauses (give the range, e.g. 30 s–3 min) and the items in a random order each time.
   - c) Each item gets its own random time within the range of question 1, independently of the other items.

4. **Items stopped on 20 Nov 2026.** After this change is installed, what happens to the Amazon items that are already stopped?
   - a) They stay stopped until you choose **Refresh price** on each one (current rule).
   - b) They are all restarted once automatically and checked at their next scheduled time.
   - c) The Amazon tab warning gets a **Restart all** action that restarts every stopped item at once.
