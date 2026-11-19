# Issue #165 Use all space to show item list when the screen is reduced

On reduced screens, it is a shame that full space is not used. as you can see on the screenshot, the yellow part is not used when the screen is reduced.
It should be nice that the page is fully used when the view size is reduced.

<img width="775" height="806" alt="image" src="https://github.com/user-attachments/assets/5bc167ea-05c4-482a-be93-f473ce5f13b6" />

## Comments by trusted authors

### redjhawk at 2026-11-19T21:02:30Z

Below 672 px the cards already span the full screen width> no, it doesnt.  app-content" keeps a padding at right and at left.  Then, cds--tab-content applies more padding. 
These two paddings make 48 px at each side ! in a big screen, it is nice. However, in little screens, this is a huge lost.
In smaller screens, those paddings must dissapear 

padding must dissapear as soon sa the inside card layout change


