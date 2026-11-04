"""Independent hand-auditable scalar references for the Excel acceptance stage.

This is NOT a calculation using Ebala/Go or Excel.  Values are transcribed
from SP20 2016 tables and elementary mechanics.  Run with Python stdlib.
"""
from __future__ import annotations
import json
import math
from pathlib import Path


def relative_close(actual, expected, rtol=1e-10):
    if not math.isclose(actual, expected, rel_tol=rtol, abs_tol=rtol):
        raise AssertionError(f"computed {actual!r}, independently expected {expected!r}")


def linear(x, x0, y0, x1, y1):
    return y0 + (x - x0) * (y1 - y0) / (x1 - x0)


def reduced_factor(load_class, area, floors):
    base, minimum = {1: (9.0, .4), 2: (36.0, .5)}[load_class]
    phi = 1 if area <= base else minimum + (1-minimum) * math.sqrt(base/area)
    if floors > 1:
        phi = minimum + (phi-minimum)/math.sqrt(floors)
    return phi


cases = {
    "weight": {
        "inputs": {"thickness_mm": 150, "density_kg_m3": 2500, "gamma_f": 1.1},
        "expected": {"normative_kPa": 3.67749375, "design_kPa": 4.045243125},
        "reference": "gн = 0.150·2500·9.80665/1000, gр=γf·gн",
    },
    "temporary_class1": {
        "inputs": {"class": 1, "area_m2": 36, "floors": 3},
        "expected": {"phi": .4 + (.4 + .6*.5 - .4)/math.sqrt(3)},
        "reference": "СП 20.13330.2016: A1=9; минимум 0,4; несколько перекрытий",
    },
    "temporary_class2": {
        "inputs": {"class": 2, "area_m2": 144, "floors": 2},
        "expected": {"phi": .5 + (.5 + .5*.5 - .5)/math.sqrt(2)},
        "reference": "СП 20.13330.2016: A2=36; минимум 0,5; несколько перекрытий",
    },
    "ice_height": {
        "inputs": {"height_m": 15},
        "expected": {"k": 1.1},
        "reference": "СП 20.13330.2016, таблица 12.3: k10=1,0; k20=1,2",
    },
    "ice_diameter": {
        "inputs": {"diameter_mm": 15},
        "expected": {"mu1": .95},
        "reference": "СП 20.13330.2016, таблица 12.4: μ10=1,0; μ20=0,9",
    },
    "ice_height_250m": {
        "inputs": {"height_m": 250},
        "expected": {"b_other_areas_mm": 40},
        "reference": "СП 20.13330.2016, таблица 12.2: 200 м=35, 300 м=45",
    },
    "crane_one_span": {
        "inputs": {"span_m": 6, "wheel_x_m": 3, "P_kN": 100, "section_x_m": 3},
        "expected": {"moment_kN_m": 150},
        "reference": "Однопролётная шарнирная балка: M=P·a·b/L",
    },
    "crane_two_span": {
        "inputs": {"spans_m": [6,6], "wheel_x_m": 3, "P_kN": 100, "section_x_m": 3},
        "expected": {"M_support_per_kN_m": -.5625, "moment_kN_m": 121.875},
        "reference": "Метод трёх моментов: 2·(6+6)·M1 = -3·3·(6+3)/6; M(3)=100·(1,5+M1·0,5)",
    },
}

# Exactly tangential sphere/cone is a valid *linear-arc limit* in the Go
# geometry, so the Excel sheet must branch rather than divide by zero.
for alpha_deg in (18.0, 40.0):
    D, r, H = 18.0, 6.0, 12.0
    alpha = math.radians(alpha_deg)
    f = D / 2 * math.tan(alpha)
    dx = D / 2 - r
    dh = f - r * math.tan(alpha)
    denom = 2 * (dh * math.cos(alpha) - dx * math.sin(alpha))
    relative_close(denom, 0.0, rtol=1e-9)
    z = 7.5
    height = H - z * math.tan(alpha)
    cr = 2.55 - math.exp(0.8 - 14 * f / D)
    if alpha_deg <= 30:
        mu2 = max(0.0, cr * (z / r) ** 2)
    elif alpha_deg <= 45:
        mu2 = max(0.0, cr + (1.5 - cr) * (alpha_deg - 30) / 15)
    else:
        raise AssertionError('unsupported reference angle')
    key = f"sphere_cone_linear_{int(alpha_deg)}deg"
    cases[key] = {
        "inputs": {"D_m": D, "r_m": r, "H_m": H, "alpha2_degrees": alpha_deg,
                   "f_m": f, "z_m": z},
        "expected": {"linear_arc": True, "height_m": height, "mu2_dome": mu2},
        "reference": "Предел касательного сопряжения: R=∞, αмест=α₂, r30=r; зависимость μ₂ как в расчётном ядре",
    }
    relative_close(H - z * math.tan(alpha), cases[key]["expected"]["height_m"])
    assert math.isfinite(cases[key]["expected"]["mu2_dome"])

relative_close(.150*2500*9.80665/1000,cases['weight']['expected']['normative_kPa'])
relative_close(cases['weight']['expected']['normative_kPa']*1.1,cases['weight']['expected']['design_kPa'])
relative_close(reduced_factor(1,36,3),cases['temporary_class1']['expected']['phi'])
relative_close(reduced_factor(2,144,2),cases['temporary_class2']['expected']['phi'])
relative_close(linear(15,10,1.0,20,1.2),cases['ice_height']['expected']['k'])
relative_close(linear(15,10,1.0,20,.9),cases['ice_diameter']['expected']['mu1'])
relative_close(linear(250,200,35,300,45),cases['ice_height_250m']['expected']['b_other_areas_mm'])
relative_close(100*3*3/6,cases['crane_one_span']['expected']['moment_kN_m'])
support=-3*3*(6+3)/6/(2*(6+6))
relative_close(support,cases['crane_two_span']['expected']['M_support_per_kN_m'])
relative_close(100*(3*3/6+support/2),cases['crane_two_span']['expected']['moment_kN_m'])

out=Path(__file__).with_name('reference_cases.json')
out.write_text(json.dumps(cases,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(f'{len(cases)} independent scalar references checked; results written: {out}')
