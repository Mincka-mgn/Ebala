from pathlib import Path
p=Path('/mnt/data/ebala063_work/wind_ui_windows.go')
s=p.read_text()
a=s.index('func windDrawZoneDiagram(')
b=s.index('func windPaint(',a)
new=r'''func windDrawZoneDiagram(dc uintptr, x, y, w, h int, v *WindVariant, active *WindZoneCalculation) {
    if v == nil || w < 180 || h < 130 { return }
    roof := active != nil && active.Surface == "roof"
    endOnly := active != nil && active.Surface != "roof" && (active.Code == "D" || active.Code == "E")
    heading := "План здания. " + v.Name
    if v.ID == "W" { heading = v.Name }
    windDrawString(dc, x+6, y+4, heading, 0x222222)
    if v.ID == "W" {
        left, top, pw := x+20, y+75, w-60
        cuts := []float64{0, v.AlongM}
        for _, z := range v.Zones { cuts=append(cuts,z.From,z.To) }
        ax:=windAxis(cuts,left,pw)
        windStroke(dc,[][2]int{{left,top},{left+pw,top}},0x222222,2)
        for _,z:=range v.Zones {
            from,to:=ax.px(z.From),ax.px(z.To)
            col,thick:=uintptr(0x777777),2
            if active!=nil && active.Code==z.Code {col,thick=0xA05020,4}
            windStroke(dc,[][2]int{{from,top},{to,top}},col,thick)
            windDrawString(dc,(from+to)/2-5,top-25,z.Code,0x222222)
            yy:=top+13
            windStroke(dc,[][2]int{{from,yy-4},{from,yy},{to,yy},{to,yy-4}},0x777777,1)
            windDrawString(dc,from,yy+9,decimalInput(z.To-z.From,2)+" м",0x333333)
        }
        windDrawString(dc,left+pw/2-22,top+47,"Ветер",0x222222)
        windStroke(dc,[][2]int{{left+pw/2+23,top+55},{left+pw/2+69,top+55},{left+pw/2+62,top+50},{left+pw/2+69,top+55},{left+pw/2+62,top+60}},0x202020,2)
        return
    }
    left,top:=x+41,y+58
    pw,ph:=w-109,h-133
    if ph>168 {ph=168};if ph<50 {ph=50};if pw<85 {pw=85}
    alt:=""
    if roof {alt=active.Alternative}
    boxes,xCuts,yCuts:=windDrawingRects(*v,roof,alt)
    ax,ay:=windAxis(xCuts,left,pw),windAxis(yCuts,top,ph)
    windStroke(dc,[][2]int{{left,top},{left+pw,top},{left+pw,top+ph},{left,top+ph},{left,top}},0x222222,2)
    if !endOnly {
        for _,r:=range boxes {
            x1,x2:=ax.px(r.x0),ax.px(r.x1)
            y1,y2:=ay.px(r.y0),ay.px(r.y1)
            col,thick:=uintptr(0x888888),1
            if active!=nil && active.Code==r.code {col,thick=0xA05020,3}
            if roof {
                windStroke(dc,[][2]int{{x1,y1},{x2,y1},{x2,y2},{x1,y2},{x1,y1}},col,thick)
                if x2-x1>=17 && y2-y1>=17 {
                    windDrawString(dc,(x1+x2)/2-4,(y1+y2)/2-8,r.code,0x222222)
                } else {
                    // A label outside the narrow patch with a short leader.
                    tx:=x1+2
                    if x2-x1<17 {tx=left-14}
                    windDrawString(dc,tx,(y1+y2)/2-7,r.code,0x222222)
                    if x2-x1<17 {windStroke(dc,[][2]int{{tx+9,(y1+y2)/2},{x1+2,(y1+y2)/2}},0x777777,1)}
                }
            } else if v.ID=="RIDGE" {
                // A/B/C are line segments along the side-wall contours.
                windStroke(dc,[][2]int{{x1,top},{x2,top}},col,thick)
                windStroke(dc,[][2]int{{x1,top+ph},{x2,top+ph}},col,thick)
                windStroke(dc,[][2]int{{x1,top-4},{x1,top+4}},0x777777,1)
                windStroke(dc,[][2]int{{x2,top-4},{x2,top+4}},0x777777,1)
                windDrawString(dc,(x1+x2)/2-4,top+5,r.code,0x222222)
            } else {
                windStroke(dc,[][2]int{{left,y1},{left,y2}},col,thick)
                windStroke(dc,[][2]int{{left+pw,y1},{left+pw,y2}},col,thick)
                windStroke(dc,[][2]int{{left-4,y1},{left+4,y1}},0x777777,1)
                windStroke(dc,[][2]int{{left-4,y2},{left+4,y2}},0x777777,1)
                windDrawString(dc,left+4,(y1+y2)/2-8,r.code,0x222222)
            }
        }
    }
    if !roof && endOnly {
        // End-wall result: the plan identifies only the requested end face.
        if v.ID=="RIDGE" {
            edge:=left;if active.Code=="E" {edge=left+pw}
            windStroke(dc,[][2]int{{edge,top},{edge,top+ph}},0xA05020,4)
            if active.Code=="D" {windDrawString(dc,edge-16,top+ph/2-12,"D",0x222222)} else {windDrawString(dc,edge+4,top+ph/2-12,"E",0x222222)}
        } else {
            edge:=top+ph;if active.Code=="E" {edge=top}
            windStroke(dc,[][2]int{{left,edge},{left+pw,edge}},0xA05020,4)
            windDrawString(dc,left+pw/2-5,edge+7,active.Code,0x222222)
        }
    }
    // All building plans include a clearly identifiable ridge. It does not
    // exist when h==0. Never draw a dashed ridge or an axis-break mark.
    if active!=nil && active.Calculation.Input.RoofRise>0 {
        mid:=top+ph/2
        windStroke(dc,[][2]int{{left,mid},{left+pw,mid}},0x202020,3)
        windDrawString(dc,left+pw-56,mid-20,"Конёк",0x222222)
    }
    // Wind arrow and its caption form one grouped, centred symbol.
    if v.ID=="RIDGE" {
        cy:=top+ph/2
        windStroke(dc,[][2]int{{left-29,cy},{left-8,cy},{left-16,cy-5},{left-8,cy},{left-16,cy+5}},0x202020,2)
        windDrawString(dc,left-37,cy+8,"Ветер",0x222222)
    } else {
        cx:=left+pw/2
        windStroke(dc,[][2]int{{cx,top+ph+39},{cx,top+ph+8},{cx-6,top+ph+16},{cx,top+ph+8},{cx+6,top+ph+16}},0x202020,2)
        windDrawString(dc,cx+11,top+ph+15,"Ветер",0x222222)
    }
    if endOnly { return }
    // Dimension values are mandatory even for narrow physical strips.
    for k:=0;k<len(xCuts)-1;k++ {
        a,b:=ax.px(xCuts[k]),ax.px(xCuts[k+1]); yy:=top-9
        windStroke(dc,[][2]int{{a,yy+3},{a,yy},{b,yy},{b,yy+3}},0x777777,1)
        tx:=(a+b)/2-14;ty:=yy-19
        if b-a<55 { tx=a;ty=yy-36-14*(k%2) }
        windDrawString(dc,tx,ty,decimalInput(xCuts[k+1]-xCuts[k],2)+" м",0x333333)
    }
    lastY:=top-19
    for k:=0;k<len(yCuts)-1;k++ {
        a,b:=ay.px(yCuts[k]),ay.px(yCuts[k+1]);xx:=left+pw+9
        windStroke(dc,[][2]int{{xx-3,a},{xx,a},{xx,b},{xx-3,b}},0x777777,1)
        mid:=(a+b)/2;yy:=mid-8
        if yy<lastY+17 {yy=lastY+17}
        windStroke(dc,[][2]int{{xx+1,mid},{xx+5,yy+7}},0x777777,1)
        windDrawString(dc,xx+8,yy,decimalInput(yCuts[k+1]-yCuts[k],2)+" м",0x333333)
        lastY=yy
    }
    if roof {
        names:=[]string{}
        for _,z:=range v.Zones {if z.Surface=="roof" && z.Alternative==alt {seen:=false;for _,q:=range names{if q==z.Code{seen=true;break}};if !seen{names=append(names,z.Code)}}}
        windDrawString(dc,x+5,top+ph+54,"Зоны кровли: "+strings.Join(names,", "),0x333333)
    }
}
'''
s=s[:a]+new+s[b:]
p.write_text(s)
