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


### redjhawk at 2026-11-19T21:08:22Z

1) C. Both A and B.> parent components of the cards use margin. That margin must be removed
2) B
3) C
4) D: all screen should use available space

### redjhawk at 2026-11-19T21:31:44Z

Continúa el desarrollo!

### redjhawk at 2026-11-19T22:17:11Z

relanza

### redjhawk at 2026-11-20T05:49:04Z

Can you continue please?

### redjhawk at 2026-11-20T16:01:56Z

@claude  do whatever you have to do. fix whatever you have to fix. All I want is that it works

### redjhawk at 2026-11-20T17:29:57Z

can you continue ?


### redjhawk at 2026-11-20T17:44:59Z

continue

### redjhawk at 2026-11-20T17:47:30Z

@claude  en la rama principal hay una corrección para el error que estás teniendo al desarrollar. Haz primero un rebase de main y sigue después desarrollando

### redjhawk at 2026-11-20T18:36:03Z

@claude 
continue

